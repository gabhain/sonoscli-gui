package sonos

import (
	"context"
	"strconv"
)

func (c *Client) GetEQ(ctx context.Context, eqType string) (int, error) {
	resp, err := c.soapCall(ctx, controlRenderingControl, urnRenderingControl, "GetEQ", map[string]string{
		"InstanceID": "0",
		"EQType":     eqType,
	})
	if err != nil {
		return 0, err
	}
	v, _ := strconv.Atoi(resp["CurrentValue"])
	return v, nil
}

func (c *Client) SetEQ(ctx context.Context, eqType string, value int) error {
	_, err := c.soapCall(ctx, controlRenderingControl, urnRenderingControl, "SetEQ", map[string]string{
		"InstanceID":   "0",
		"EQType":       eqType,
		"DesiredValue": strconv.Itoa(value),
	})
	return err
}

func (c *Client) GetLoudness(ctx context.Context) (bool, error) {
	resp, err := c.soapCall(ctx, controlRenderingControl, urnRenderingControl, "GetLoudness", map[string]string{
		"InstanceID": "0",
		"Channel":    "Master",
	})
	if err != nil {
		return false, err
	}
	return resp["CurrentLoudness"] == "1", nil
}

func (c *Client) SetLoudness(ctx context.Context, loudness bool) error {
	val := "0"
	if loudness {
		val = "1"
	}
	_, err := c.soapCall(ctx, controlRenderingControl, urnRenderingControl, "SetLoudness", map[string]string{
		"InstanceID":      "0",
		"Channel":         "Master",
		"DesiredLoudness": val,
	})
	return err
}

// NightMode and Speech Enhancement are often specific EQ types or separate methods
// In modern Sonos, they are often DialogLevel and NightMode

func (c *Client) GetNightMode(ctx context.Context) (bool, error) {
	v, err := c.GetEQ(ctx, "NightMode")
	return v == 1, err
}

func (c *Client) SetNightMode(ctx context.Context, on bool) error {
	v := 0
	if on { v = 1 }
	return c.SetEQ(ctx, "NightMode", v)
}

func (c *Client) GetSpeechEnhancement(ctx context.Context) (bool, error) {
	v, err := c.GetEQ(ctx, "DialogLevel")
	return v == 1, err
}

func (c *Client) SetSpeechEnhancement(ctx context.Context, on bool) error {
	v := 0
	if on { v = 1 }
	return c.SetEQ(ctx, "DialogLevel", v)
}
