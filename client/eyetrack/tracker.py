import os
import signal
import sys

os.environ["OPENCV_LOG_LEVEL"] = "SILENT"
os.environ["OPENCV_VIDEOIO_DEBUG"] = "0"

try:
    signal.signal(signal.SIGPIPE, signal.SIG_DFL)
except Exception:
    pass

import cv2
import json
import numpy as np
import argparse
import base64
import time

parser = argparse.ArgumentParser()
parser.add_argument('--camera', type=int, default=0)
parser.add_argument('--preview', type=int, default=0)
args = parser.parse_args()

def send_data(data_dict):
    try:
        print(json.dumps(data_dict), flush=True)
    except BrokenPipeError:
        sys.exit(0)
    except Exception:
        sys.exit(0)

def print_error(msg):
    img = np.zeros((240, 320, 3), dtype=np.uint8)
    words = msg.split()
    line = ""
    lines = []

    for word in words:
        if len(line) + len(word) < 45:
            line += word + " "
        else:
            lines.append(line)
            line = word + " "
    lines.append(line)

    y0, dy = 120, 20
    if len(lines) > 1:
        y0 = 120 - ((len(lines) // 2) * dy)

    for i, l in enumerate(lines):
        cv2.putText(img, l.strip(), (10, y0 + i * dy), cv2.FONT_HERSHEY_SIMPLEX, 0.4, (0, 0, 255), 1, cv2.LINE_AA)

    _, buffer = cv2.imencode('.jpg', img, [cv2.IMWRITE_JPEG_QUALITY, 60])
    send_data({"x": 0.5, "y": 0.5, "frame": base64.b64encode(buffer).decode('utf-8')})

try:
    import mediapipe as mp
    mp_face_mesh = mp.solutions.face_mesh
except Exception as e:
    msg = f"MP Err: {str(e)} ({sys.executable})"
    while True:
        if args.preview == 1:
            print_error(msg)
        else:
            send_data({"x": 0.5, "y": 0.5})
        time.sleep(1)

RIGHT_IRIS = [469, 470, 471, 472]
LEFT_IRIS = [474, 475, 476, 477]

cap = cv2.VideoCapture(args.camera)
if not cap.isOpened():
    while True:
        if args.preview == 1:
            print_error(f"Cannot open camera {args.camera}!")
        else:
            send_data({"x": 0.5, "y": 0.5})
        time.sleep(1)

class EMAFilter:
    def __init__(self, alpha=0.15):
        self.alpha = alpha
        self.val = None

    def update(self, new_val):
        if self.val is None:
            self.val = new_val
        else:
            self.val = self.alpha * new_val + (1 - self.alpha) * self.val
        return self.val

filter_x = EMAFilter(alpha=0.15)
filter_y = EMAFilter(alpha=0.15)

with mp_face_mesh.FaceMesh(max_num_faces=1, refine_landmarks=True, min_detection_confidence=0.5, min_tracking_confidence=0.5) as face_mesh:
    while cap.isOpened():
        success, image = cap.read()
        if not success:
            time.sleep(0.1)
            continue

        image = cv2.flip(image, 1)
        img_h, img_w = image.shape[:2]
        results = face_mesh.process(cv2.cvtColor(image, cv2.COLOR_BGR2RGB))

        gaze_x, gaze_y = 0.5, 0.5

        if results.multi_face_landmarks:
            mesh_points = np.array([np.multiply([p.x, p.y], [img_w, img_h]).astype(int) for p in results.multi_face_landmarks[0].landmark])
            (l_cx, l_cy), _ = cv2.minEnclosingCircle(mesh_points[LEFT_IRIS])
            (r_cx, r_cy), _ = cv2.minEnclosingCircle(mesh_points[RIGHT_IRIS])

            eye_left = mesh_points[33][0]
            eye_right = mesh_points[133][0]
            eye_width = eye_right - eye_left

            if eye_width > 0:
                raw_x = (r_cx - eye_left) / eye_width
                raw_y = ((l_cy + r_cy) / 2.0) / img_h
                gaze_x = filter_x.update(raw_x)
                gaze_y = filter_y.update(raw_y)

            if args.preview == 1:
                cv2.circle(image, (int(l_cx), int(l_cy)), 3, (0, 255, 0), -1)
                cv2.circle(image, (int(r_cx), int(r_cy)), 3, (0, 255, 0), -1)

        out_dict = {"x": gaze_x, "y": gaze_y}

        if args.preview == 1:
            preview_img = cv2.resize(image, (320, 240))
            _, buffer = cv2.imencode('.jpg', preview_img, [cv2.IMWRITE_JPEG_QUALITY, 60])
            out_dict["frame"] = base64.b64encode(buffer).decode('utf-8')

        send_data(out_dict)

cap.release()