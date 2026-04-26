package sonos

import (
	"context"
	"strconv"
)

type BrowseResponse struct {
	Result         string
	NumberReturned int
	TotalMatches   int
	UpdateID       int
}

func (c *Client) Browse(ctx context.Context, objectID string, start, count int) (BrowseResponse, error) {
	resp, err := c.soapCall(ctx, controlContentDirectory, urnContentDirectory, "Browse", map[string]string{
		"ObjectID":       objectID,
		"BrowseFlag":     "BrowseDirectChildren",
		"Filter":         "*",
		"StartingIndex":  strconv.Itoa(start),
		"RequestedCount": strconv.Itoa(count),
		"SortCriteria":   "",
	})
	if err != nil {
		return BrowseResponse{}, err
	}

	out := BrowseResponse{Result: resp["Result"]}
	if v, err := strconv.Atoi(resp["NumberReturned"]); err == nil {
		out.NumberReturned = v
	}
	if v, err := strconv.Atoi(resp["TotalMatches"]); err == nil {
		out.TotalMatches = v
	}
	if v, err := strconv.Atoi(resp["UpdateID"]); err == nil {
		out.UpdateID = v
	}
	return out, nil
}
