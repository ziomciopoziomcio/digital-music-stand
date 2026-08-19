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

	deletedIDs, err := dbMgr.GetDeletedScoreIDs()
	if err == nil {
		for _, id := range deletedIDs {
			log.Printf("Pushing deletion for score %s to cloud...", id)
			_, err := scoreClient.DeleteScore(ctx, &scorepb.DeleteScoreRequest{Id: id})
			if err != nil {
				log.Printf("Failed to delete score %s on server: %v", id, err)
			} else {
				_ = dbMgr.HardDeleteScore(id)
			}
		}
	}

	remoteResp, err := scoreClient.ListMyScores(ctx, &scorepb.ListMyScoresRequest{})
	if err != nil {
		return fmt.Errorf("failed to list remote scores: %w", err)
	}

	remoteMap := make(map[string]*scorepb.Score)
	for _, rs := range remoteResp.GetScores() {
		remoteMap[rs.Id] = rs
	}

	localScores, err := dbMgr.GetScores()
	if err != nil {
		return fmt.Errorf("failed to get local scores: %w", err)
	}

	for _, ls := range localScores {
		if ls.Checksum == "" {
			if _, existsOnServer := remoteMap[ls.ID]; existsOnServer {
				log.Printf("Updating remote score %s (%s)...", ls.Title, ls.ID)
				resp, err := scoreClient.UpdateScore(ctx, &scorepb.UpdateScoreRequest{
					Id:   ls.ID,
					Name: ls.Title,
				})
				if err != nil {
					log.Printf("Failed to update score %s on server: %v", ls.Title, err)
				} else {
					_ = dbMgr.SyncScoreFromServer(ls.ID, ls.Title, ls.FilePath, resp.Checksum, true)
				}
			} else {
				log.Printf("Pushing new score %s (%s) to cloud...", ls.Title, ls.ID)

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
					_ = dbMgr.SyncScoreFromServer(resp.Id, ls.Title, ls.FilePath, resp.Checksum, true)
				}
			}
		}
	}

	remoteResp, err = scoreClient.ListMyScores(ctx, &scorepb.ListMyScoresRequest{})
	if err != nil {
		return fmt.Errorf("failed to list remote scores after push: %w", err)
	}

	remoteMap = make(map[string]*scorepb.Score)
	for _, rs := range remoteResp.GetScores() {
		remoteMap[rs.Id] = rs
	}

	localScoresMap := make(map[string]localdb.Score)
	localScores, _ = dbMgr.GetScores()
	for _, ls := range localScores {
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

			err = dbMgr.SyncScoreFromServer(rs.Id, rs.Name, absFilePath, rs.Checksum, rs.IsOwner)
			if err != nil {
				log.Printf("Failed to save synced score to local db: %v", err)
			}
		}
	}

	for _, ls := range localScores {
		if ls.Checksum != "" {
			if _, existsOnServer := remoteMap[ls.ID]; !existsOnServer {
				log.Printf("Score %s (%s) was deleted on server. Removing locally...", ls.Title, ls.ID)
				_ = dbMgr.HardDeleteScore(ls.ID)
			}
		}
	}

	return nil
}
