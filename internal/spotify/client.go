package spotify

import (
	"context"
	"encoding/base64"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"os"
	"strconv"
	"strings"
	"sync"
	"time"
)

type SearchType string

const (
	TypeTrack    SearchType = "track"
	TypeAlbum    SearchType = "album"
	TypePlaylist SearchType = "playlist"
	TypeShow     SearchType = "show"
	TypeEpisode  SearchType = "episode"
)

func ParseSearchType(s string) (SearchType, error) {
	switch SearchType(strings.ToLower(strings.TrimSpace(s))) {
	case TypeTrack, TypeAlbum, TypePlaylist, TypeShow, TypeEpisode:
		return SearchType(strings.ToLower(strings.TrimSpace(s))), nil
	default:
		return "", fmt.Errorf("invalid type %q (use track|album|playlist|show|episode)", s)
	}
}

type Client struct {
	ClientID     string
	ClientSecret string

	AccountsBaseURL string
	APIBaseURL      string
	HTTP            *http.Client

	mu        sync.Mutex
	token     string
	tokenType string
	expiresAt time.Time
}

func NewFromEnv(httpClient *http.Client) (*Client, error) {
	id := strings.TrimSpace(os.Getenv("SPOTIFY_CLIENT_ID"))
	secret := strings.TrimSpace(os.Getenv("SPOTIFY_CLIENT_SECRET"))
	if id == "" || secret == "" {
		return nil, errors.New("missing SPOTIFY_CLIENT_ID / SPOTIFY_CLIENT_SECRET (Spotify Web API client credentials)")
	}
	return New(id, secret, httpClient), nil
}

func New(clientID, clientSecret string, httpClient *http.Client) *Client {
	if httpClient == nil {
		httpClient = &http.Client{Timeout: 10 * time.Second}
	}
	return &Client{
		ClientID:        clientID,
		ClientSecret:    clientSecret,
		HTTP:            httpClient,
		tokenType:       "Bearer",
		AccountsBaseURL: "https://accounts.spotify.com",
		APIBaseURL:      "https://api.spotify.com",
	}
}

type Result struct {
	Type     SearchType `json:"type"`
	ID       string     `json:"id"`
	URI      string     `json:"uri"`
	URL      string     `json:"url"`
	Title    string     `json:"title"`
	Subtitle string     `json:"subtitle,omitempty"`
}

func (c *Client) Search(ctx context.Context, query string, typ SearchType, limit int, market string) ([]Result, error) {
	query = strings.TrimSpace(query)
	if query == "" {
		return nil, errors.New("query is required")
	}
	if limit <= 0 {
		limit = 10
	}
	if limit > 50 {
		limit = 50
	}
	market = strings.TrimSpace(market)

	if err := c.ensureToken(ctx); err != nil {
		return nil, err
	}

	u, err := url.Parse(c.APIBaseURL + "/v1/search")
	if err != nil {
		return nil, err
	}
	q := u.Query()
	q.Set("q", query)
	q.Set("type", string(typ))
	q.Set("limit", strconv.Itoa(limit))
	if market != "" {
		q.Set("market", market)
	}
	u.RawQuery = q.Encode()

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, u.String(), nil)
	if err != nil {
		return nil, err
	}
	req.Header.Set("Authorization", c.tokenType+" "+c.token)

	resp, err := c.HTTP.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		body, _ := io.ReadAll(io.LimitReader(resp.Body, 4096))
		return nil, fmt.Errorf("spotify search: %s: %s", resp.Status, strings.TrimSpace(string(body)))
	}

	b, err := io.ReadAll(io.LimitReader(resp.Body, 4<<20))
	if err != nil {
		return nil, err
	}

	switch typ {
	case TypeTrack:
		var sr struct {
			Tracks struct {
				Items []struct {
					ID           string `json:"id"`
					Name         string `json:"name"`
					URI          string `json:"uri"`
					ExternalURLs struct {
						Spotify string `json:"spotify"`
					} `json:"external_urls"`
					Artists []struct {
						Name string `json:"name"`
					} `json:"artists"`
					Album struct {
						Name string `json:"name"`
					} `json:"album"`
				} `json:"items"`
			} `json:"tracks"`
		}
		if err := json.Unmarshal(b, &sr); err != nil {
			return nil, err
		}
		out := make([]Result, 0, len(sr.Tracks.Items))
		for _, it := range sr.Tracks.Items {
			artists := make([]string, 0, len(it.Artists))
			for _, a := range it.Artists {
				if a.Name != "" {
					artists = append(artists, a.Name)
				}
			}
			sub := strings.Join(artists, ", ")
			if it.Album.Name != "" && sub != "" {
				sub = sub + " — " + it.Album.Name
			} else if it.Album.Name != "" {
				sub = it.Album.Name
			}
			out = append(out, Result{
				Type:     TypeTrack,
				ID:       it.ID,
				URI:      it.URI,
				URL:      it.ExternalURLs.Spotify,
				Title:    it.Name,
				Subtitle: sub,
			})
		}
		return out, nil
	case TypeAlbum:
		var sr struct {
			Albums struct {
				Items []struct {
					ID           string `json:"id"`
					Name         string `json:"name"`
					URI          string `json:"uri"`
					ExternalURLs struct {
						Spotify string `json:"spotify"`
					} `json:"external_urls"`
					Artists []struct {
						Name string `json:"name"`
					} `json:"artists"`
				} `json:"items"`
			} `json:"albums"`
		}
		if err := json.Unmarshal(b, &sr); err != nil {
			return nil, err
		}
		out := make([]Result, 0, len(sr.Albums.Items))
		for _, it := range sr.Albums.Items {
			artists := make([]string, 0, len(it.Artists))
			for _, a := range it.Artists {
				if a.Name != "" {
					artists = append(artists, a.Name)
				}
			}
			out = append(out, Result{
				Type:     TypeAlbum,
				ID:       it.ID,
				URI:      it.URI,
				URL:      it.ExternalURLs.Spotify,
				Title:    it.Name,
				Subtitle: strings.Join(artists, ", "),
			})
		}
		return out, nil
	case TypePlaylist:
		var sr struct {
			Playlists struct {
				Items []struct {
					ID           string `json:"id"`
					Name         string `json:"name"`
					URI          string `json:"uri"`
					ExternalURLs struct {
						Spotify string `json:"spotify"`
					} `json:"external_urls"`
					Owner struct {
						DisplayName string `json:"display_name"`
					} `json:"owner"`
					Tracks struct {
						Total int `json:"total"`
					} `json:"tracks"`
				} `json:"items"`
			} `json:"playlists"`
		}
		if err := json.Unmarshal(b, &sr); err != nil {
			return nil, err
		}
		out := make([]Result, 0, len(sr.Playlists.Items))
		for _, it := range sr.Playlists.Items {
			sub := strings.TrimSpace(it.Owner.DisplayName)
			if it.Tracks.Total > 0 {
				if sub != "" {
					sub = fmt.Sprintf("%s — %d tracks", sub, it.Tracks.Total)
				} else {
					sub = fmt.Sprintf("%d tracks", it.Tracks.Total)
				}
			}
			out = append(out, Result{
				Type:     TypePlaylist,
				ID:       it.ID,
				URI:      it.URI,
				URL:      it.ExternalURLs.Spotify,
				Title:    it.Name,
				Subtitle: sub,
			})
		}
		return out, nil
	case TypeShow:
		var sr struct {
			Shows struct {
				Items []struct {
					ID           string `json:"id"`
					Name         string `json:"name"`
					URI          string `json:"uri"`
					ExternalURLs struct {
						Spotify string `json:"spotify"`
					} `json:"external_urls"`
					Publisher string `json:"publisher"`
				} `json:"items"`
			} `json:"shows"`
		}
		if err := json.Unmarshal(b, &sr); err != nil {
			return nil, err
		}
		out := make([]Result, 0, len(sr.Shows.Items))
		for _, it := range sr.Shows.Items {
			out = append(out, Result{
				Type:     TypeShow,
				ID:       it.ID,
				URI:      it.URI,
				URL:      it.ExternalURLs.Spotify,
				Title:    it.Name,
				Subtitle: strings.TrimSpace(it.Publisher),
			})
		}
		return out, nil
	case TypeEpisode:
		var sr struct {
			Episodes struct {
				Items []struct {
					ID           string `json:"id"`
					Name         string `json:"name"`
					URI          string `json:"uri"`
					ExternalURLs struct {
						Spotify string `json:"spotify"`
					} `json:"external_urls"`
					Show struct {
						Name string `json:"name"`
					} `json:"show"`
				} `json:"items"`
			} `json:"episodes"`
		}
		if err := json.Unmarshal(b, &sr); err != nil {
			return nil, err
		}
		out := make([]Result, 0, len(sr.Episodes.Items))
		for _, it := range sr.Episodes.Items {
			out = append(out, Result{
				Type:     TypeEpisode,
				ID:       it.ID,
				URI:      it.URI,
				URL:      it.ExternalURLs.Spotify,
				Title:    it.Name,
				Subtitle: strings.TrimSpace(it.Show.Name),
			})
		}
		return out, nil
	default:
		return nil, fmt.Errorf("unsupported type: %s", typ)
	}
}

func (c *Client) ensureToken(ctx context.Context) error {
	c.mu.Lock()
	if c.token != "" && time.Until(c.expiresAt) > 30*time.Second {
		c.mu.Unlock()
		return nil
	}
	c.mu.Unlock()

	token, tokenType, expiresIn, err := c.fetchClientCredentialsToken(ctx)
	if err != nil {
		return err
	}

	c.mu.Lock()
	defer c.mu.Unlock()
	c.token = token
	if strings.TrimSpace(tokenType) != "" {
		c.tokenType = strings.TrimSpace(tokenType)
	}
	c.expiresAt = time.Now().Add(time.Duration(expiresIn) * time.Second)
	return nil
}

func (c *Client) fetchClientCredentialsToken(ctx context.Context) (token string, tokenType string, expiresIn int, err error) {
	u, err := url.Parse(c.AccountsBaseURL + "/api/token")
	if err != nil {
		return "", "", 0, err
	}
	form := url.Values{}
	form.Set("grant_type", "client_credentials")

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, u.String(), strings.NewReader(form.Encode()))
	if err != nil {
		return "", "", 0, err
	}
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	req.Header.Set("Authorization", "Basic "+base64.StdEncoding.EncodeToString([]byte(c.ClientID+":"+c.ClientSecret)))

	resp, err := c.HTTP.Do(req)
	if err != nil {
		return "", "", 0, err
	}
	defer resp.Body.Close()
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		body, _ := io.ReadAll(io.LimitReader(resp.Body, 4096))
		return "", "", 0, fmt.Errorf("spotify token: %s: %s", resp.Status, strings.TrimSpace(string(body)))
	}

	var tr struct {
		AccessToken string `json:"access_token"`
		TokenType   string `json:"token_type"`
		ExpiresIn   int    `json:"expires_in"`
	}
	b, err := io.ReadAll(io.LimitReader(resp.Body, 1<<20))
	if err != nil {
		return "", "", 0, err
	}
	if err := json.Unmarshal(b, &tr); err != nil {
		return "", "", 0, err
	}
	if tr.AccessToken == "" {
		return "", "", 0, errors.New("spotify token: empty access_token")
	}
	if tr.ExpiresIn <= 0 {
		tr.ExpiresIn = 3600
	}
	return tr.AccessToken, tr.TokenType, tr.ExpiresIn, nil
}
