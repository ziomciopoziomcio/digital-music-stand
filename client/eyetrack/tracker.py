import cv2
import json
import numpy as np
import argparse
import base64
import time
import sys

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
    cv2.putText(img, msg, (10, 120), cv2.FONT_HERSHEY_SIMPLEX, 0.4, (0, 0, 255), 1, cv2.LINE_AA)
    _, buffer = cv2.imencode('.jpg', img, [cv2.IMWRITE_JPEG_QUALITY, 60])
    send_data({"x": 0.5, "y": 0.5, "frame": base64.b64encode(buffer).decode('utf-8')})

try:
    import mediapipe as mp
    mp_face_mesh = mp.solutions.face_mesh
except Exception as e:
    while True:
        if args.preview == 1:
            print_error(f"Init Error: {str(e)[:40]}")
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
                iris_ratio_x = (r_cx - eye_left) / eye_width
                gaze_x = np.clip((iris_ratio_x - 0.3) / 0.4, 0.0, 1.0)
                gaze_y = np.clip(((l_cy + r_cy) / 2.0) / img_h, 0.0, 1.0)

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