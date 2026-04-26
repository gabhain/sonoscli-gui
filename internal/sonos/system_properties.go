package sonos

import (
	"context"
	"errors"
	"strings"
)

func (c *Client) GetString(ctx context.Context, variableName string) (string, error) {
	variableName = strings.TrimSpace(variableName)
	if variableName == "" {
		return "", errors.New("variableName is required")
	}
	resp, err := c.soapCall(ctx, controlSystemProperties, urnSystemProperties, "GetString", map[string]string{
		"VariableName": variableName,
	})
	if err != nil {
		return "", err
	}
	v := strings.TrimSpace(resp["StringValue"])
	return v, nil
}
