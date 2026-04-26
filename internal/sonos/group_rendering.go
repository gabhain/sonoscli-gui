package sonos

import (
	"context"
	"strconv"
)

func (c *Client) SnapshotGroupVolume(ctx context.Context) error {
	_, err := c.soapCall(ctx, controlGroupRendering, urnGroupRenderingControl, "SnapshotGroupVolume", map[string]string{
		"InstanceID": "0",
	})
	return err
}

func (c *Client) GetGroupVolume(ctx context.Context) (int, error) {
	resp, err := c.soapCall(ctx, controlGroupRendering, urnGroupRenderingControl, "GetGroupVolume", map[string]string{
		"InstanceID": "0",
	})
	if err != nil {
		return 0, err
	}
	n, err := strconv.Atoi(resp["CurrentVolume"])
	if err != nil {
		return 0, err
	}
	return n, nil
}

func (c *Client) SetGroupVolume(ctx context.Context, volume int) error {
	if volume < 0 {
		volume = 0
	}
	if volume > 100 {
		volume = 100
	}
	// SnapshotGroupVolume first (Sonos convention).
	_ = c.SnapshotGroupVolume(ctx)
	_, err := c.soapCall(ctx, controlGroupRendering, urnGroupRenderingControl, "SetGroupVolume", map[string]string{
		"InstanceID":    "0",
		"DesiredVolume": strconv.Itoa(volume),
	})
	return err
}

func (c *Client) GetGroupMute(ctx context.Context) (bool, error) {
	resp, err := c.soapCall(ctx, controlGroupRendering, urnGroupRenderingControl, "GetGroupMute", map[string]string{
		"InstanceID": "0",
	})
	if err != nil {
		return false, err
	}
	return resp["CurrentMute"] == "1", nil
}

func (c *Client) SetGroupMute(ctx context.Context, mute bool) error {
	val := "0"
	if mute {
		val = "1"
	}
	_, err := c.soapCall(ctx, controlGroupRendering, urnGroupRenderingControl, "SetGroupMute", map[string]string{
		"InstanceID":  "0",
		"DesiredMute": val,
	})
	return err
}
