package microsoft

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
)

type XboxLoginResponse struct {
	// Username is not the Minecraft username!
	Username string        `json:"username"`
	Roles    []interface{} `json:"roles"`
	// AccessToken should be used for all future requests
	AccessToken string `json:"access_token"`
	TokenType   string `json:"token_type"`
	ExpiresIn   int    `json:"expires_in"`
}

// TODO: use!
type MinecraftAPIErrorResponse struct {
	Path      string `json:"path"`
	ErrorType string `json:"errorType"`
	// ErrorCode is a string like "NOT_FOUND". The underlying json field name is "error"
	ErrorCode        string `json:"error"`
	ErrorMessage     string `json:"errorMessage"`
	DeveloperMessage string `json:"developerMessage"`
}

func (a *MinecraftAPIErrorResponse) Error() string {
	return fmt.Sprintf("%s: %s", a.ErrorType, a.ErrorMessage)
}

type GetProfileResponse struct {
	ID    string `json:"id"`
	Name  string `json:"name"`
	Skins []struct {
		ID      string `json:"id"`
		State   string `json:"state"`
		URL     string `json:"url"`
		Variant string `json:"variant"`
		Alias   string `json:"alias"`
	} `json:"skins"`
	Capes []interface{} `json:"capes"`
}

func (m *MicrosoftClient) minecraftLoginWithXbox(ctx context.Context, userHash string, token string) (*XboxLoginResponse, error) {
	body := fmt.Sprintf(`{ "identityToken": "x=%s;%s" }`, userHash, token)

	req, _ := jsonPostReqFromText(MC_API_XBOX_LOGIN, body)
	req = req.WithContext(ctx)
	res, err := m.Client.Do(req)
	if err != nil {
		return nil, err
	}

	if res.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("unexpected response with status %d (%s)", res.StatusCode, res.Status)
	}

	authRes := XboxLoginResponse{}
	err = json.NewDecoder(res.Body).Decode(&authRes)
	if err != nil {
		return nil, err
	}
	return &authRes, nil
}

func (m *MicrosoftClient) checkEntitlements(ctx context.Context, token string) error {
	req, err := http.NewRequestWithContext(ctx, "GET", MC_API_CHECK_ENTITLEMENT, nil)
	if err != nil {
		return err
	}

	if token == "" {
		return fmt.Errorf("no token provided")
	}

	req.Header.Set("Authorization", fmt.Sprintf("Bearer %s", token))
	resp, err := m.Do(req)
	if err != nil {
		return err
	}
	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("HTTP status %d", resp.StatusCode)
	}

	// we do not check the response body.
	// The getProfile request will fail if the user does not own Minecraft

	return nil
}

func (m *MicrosoftClient) getProfile(ctx context.Context, token string) (*GetProfileResponse, error) {
	req, err := http.NewRequestWithContext(ctx, "GET", MC_API_PROFILE, nil)
	if err != nil {
		return nil, err
	}

	if token == "" {
		return nil, fmt.Errorf("no token provided")
	}

	req.Header.Set("Authorization", fmt.Sprintf("Bearer %s", token))
	resp, err := m.Do(req)
	if err != nil {
		return nil, err
	}
	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("HTTP status %d", resp.StatusCode)
	}
	var profile GetProfileResponse
	if err := json.NewDecoder(resp.Body).Decode(&profile); err != nil {
		return nil, err
	}

	return &profile, nil
}
