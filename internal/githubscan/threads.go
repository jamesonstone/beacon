package githubscan

import (
	"context"
	"encoding/json"
	"fmt"
	"sort"
	"strconv"
	"strings"
	"time"

	"github.com/jamesonstone/beacon/internal/model"
)

const reviewThreadQuery = `query($owner: String!, $name: String!, $number: Int!) {
  repository(owner: $owner, name: $name) {
    pullRequest(number: $number) {
      reviewThreads(first: 100) {
        totalCount
        nodes {
          id
          isResolved
          isOutdated
          path
          line
          originalLine
          comments(first: 20) {
            totalCount
            nodes { id author { login } body url createdAt updatedAt }
          }
        }
      }
    }
  }
}`

type rawReviewComment struct {
	ID        string    `json:"id"`
	Author    rawActor  `json:"author"`
	Body      string    `json:"body"`
	URL       string    `json:"url"`
	CreatedAt time.Time `json:"createdAt"`
	UpdatedAt time.Time `json:"updatedAt"`
}

type rawReviewThread struct {
	ID           string `json:"id"`
	IsResolved   bool   `json:"isResolved"`
	IsOutdated   bool   `json:"isOutdated"`
	Path         string `json:"path"`
	Line         *int   `json:"line"`
	OriginalLine *int   `json:"originalLine"`
	Comments     struct {
		TotalCount int                `json:"totalCount"`
		Nodes      []rawReviewComment `json:"nodes"`
	} `json:"comments"`
}

type rawThreadResponse struct {
	Data struct {
		Repository struct {
			PullRequest struct {
				ReviewThreads struct {
					TotalCount int               `json:"totalCount"`
					Nodes      []rawReviewThread `json:"nodes"`
				} `json:"reviewThreads"`
			} `json:"pullRequest"`
		} `json:"repository"`
	} `json:"data"`
}

func (c Client) reviewThreadDetails(ctx context.Context, repository string, number int) ([]model.ReviewThread, int, bool, error) {
	owner, name, ok := strings.Cut(repository, "/")
	if !ok || owner == "" || name == "" {
		return nil, 0, false, fmt.Errorf("invalid GitHub repository: %s", repository)
	}
	output, err := c.run(
		ctx, "gh", "api", "graphql", "-F", "owner="+owner, "-F", "name="+name,
		"-F", "number="+strconv.Itoa(number), "-f", "query="+reviewThreadQuery,
	)
	if err != nil {
		return nil, 0, false, err
	}
	var response rawThreadResponse
	if err := json.Unmarshal(output, &response); err != nil {
		return nil, 0, false, fmt.Errorf("decode review threads: %w", err)
	}
	rawThreads := response.Data.Repository.PullRequest.ReviewThreads
	threads := make([]model.ReviewThread, 0, len(rawThreads.Nodes))
	for _, raw := range rawThreads.Nodes {
		if raw.IsResolved {
			continue
		}
		comments := make([]model.ReviewComment, 0, len(raw.Comments.Nodes))
		for _, comment := range raw.Comments.Nodes {
			body, bodyTruncated := truncateGitHubBody(comment.Body)
			comments = append(comments, model.ReviewComment{
				ID: comment.ID, Author: comment.Author.Login, Body: body,
				BodyTruncated: bodyTruncated, URL: comment.URL,
				CreatedAt: comment.CreatedAt, UpdatedAt: comment.UpdatedAt,
			})
		}
		sort.SliceStable(comments, func(i, j int) bool {
			if !comments[i].CreatedAt.Equal(comments[j].CreatedAt) {
				return comments[i].CreatedAt.Before(comments[j].CreatedAt)
			}
			return comments[i].ID < comments[j].ID
		})
		threads = append(threads, model.ReviewThread{
			ID: raw.ID, Path: raw.Path, Line: raw.Line, OriginalLine: raw.OriginalLine,
			Outdated: raw.IsOutdated, Comments: comments,
			CommentsTruncated: raw.Comments.TotalCount > len(raw.Comments.Nodes),
		})
	}
	sort.SliceStable(threads, func(i, j int) bool {
		if threads[i].Path != threads[j].Path {
			return threads[i].Path < threads[j].Path
		}
		left, right := lineValue(threads[i]), lineValue(threads[j])
		if left != right {
			return left < right
		}
		return threads[i].ID < threads[j].ID
	})
	return threads, len(threads), rawThreads.TotalCount > len(rawThreads.Nodes), nil
}

func lineValue(thread model.ReviewThread) int {
	if thread.Line != nil {
		return *thread.Line
	}
	if thread.OriginalLine != nil {
		return *thread.OriginalLine
	}
	return int(^uint(0) >> 1)
}
