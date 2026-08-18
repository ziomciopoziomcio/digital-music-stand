package network

import (
	"context"
	"fmt"
	"io"
	"log"
	"os"
	"path/filepath"

	"github.com/ziomciopoziomcio/digital-music-stand/client/localdb"
	"github.com/ziomciopoziomcio/digital-music-stand/contracts/gen/scorepb"
)

func SynchronizeScores(ctx context.Context, scoreClient scorepb.ScoreServiceClient, dbMgr *localdb.DBManager) error {
	if err := os.MkdirAll("./scores", 0755); err != nil {
		return fmt.Errorf("failed to create local scores directory: %w", err)
	}

	localScores, err := dbMgr.GetScores()
	if err != nil {
		return fmt.Errorf("failed to get local scores: %w", err)
	}

	for _, ls := range localScores {
		if ls.Checksum == "" {
			log.Printf("Pushing local score %s (%s) to cloud...", ls.Title, ls.ID)

			fileData, err := os.ReadFile(ls.FilePath)
			if err != nil {
				log.Printf("Failed to read local PDF %s: %v", ls.FilePath, err)
				continue
			}

			ext := filepath.Ext(ls.FilePath)
			if ext == "" {
				ext = ".pdf"
			}

			resp, err := scoreClient.CreateScore(ctx, &scorepb.CreateScoreRequest{
				Id:            &ls.ID,
				Name:          ls.Title,
				FileData:      fileData,
				FileExtension: ext,
			})
			if err != nil {
				log.Printf("Failed to push score %s to cloud: %v", ls.Title, err)
			} else {
				log.Printf("Score %s pushed successfully", ls.Title)
				err = dbMgr.SyncScoreFromServer(resp.Id, ls.Title, ls.FilePath, resp.Checksum)
				if err != nil {
					log.Printf("Failed to update local checksum for %s: %v", ls.Title, err)
				}
			}
		}
	}

	remoteResp, err := scoreClient.ListMyScores(ctx, &scorepb.ListMyScoresRequest{})
	if err != nil {
		return fmt.Errorf("failed to list remote scores: %w", err)
	}

	localScoresMap := make(map[string]localdb.Score)
	localScoresAfterPush, _ := dbMgr.GetScores()
	for _, ls := range localScoresAfterPush {
		localScoresMap[ls.ID] = ls
	}

	for _, rs := range remoteResp.GetScores() {
		localScore, exists := localScoresMap[rs.Id]

		if !exists || localScore.Checksum != rs.Checksum {
			log.Printf("Downloading score %s (%s) from cloud...", rs.Name, rs.Id)

			stream, err := scoreClient.DownloadScore(ctx, &scorepb.DownloadScoreRequest{
				ScoreId: rs.Id,
			})
			if err != nil {
				log.Printf("Failed to initiate download for score %s: %v", rs.Id, err)
				continue
			}

			ext := rs.FileExtension
			if ext == "" {
				ext = ".pdf"
			}

			localFilePath := filepath.Join("./scores", rs.Id+ext)
			absFilePath, err := filepath.Abs(localFilePath)
			if err != nil {
				absFilePath = localFilePath
			}

			file, err := os.Create(absFilePath)
			if err != nil {
				log.Printf("Failed to create local file %s: %v", absFilePath, err)
				continue
			}

			for {
				chunk, err := stream.Recv()
				if err == io.EOF {
					break
				}
				if err != nil {
					log.Printf("Error while downloading chunks for %s: %v", rs.Id, err)
					break
				}
				_, _ = file.Write(chunk.GetChunkData())
			}
			file.Close()

			err = dbMgr.SyncScoreFromServer(rs.Id, rs.Name, absFilePath, rs.Checksum)
			if err != nil {
				log.Printf("Failed to save synced score to local db: %v", err)
			}
		}
	}

	return nil
}
