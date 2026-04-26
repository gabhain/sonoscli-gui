package sonos

import (
	"context"
	"errors"
	"strings"
)

func (c *Client) GetHouseholdID(ctx context.Context) (string, error) {
	resp, err := c.soapCall(ctx, controlDeviceProperties, urnDeviceProperties, "GetHouseholdID", nil)
	if err != nil {
		return "", err
	}
	hh := strings.TrimSpace(resp["CurrentHouseholdID"])
	if hh == "" {
		return "", errors.New("missing CurrentHouseholdID in response")
	}
	return hh, nil
}
