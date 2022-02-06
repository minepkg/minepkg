package microsoft

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"time"
)

type xblAuthResponse struct {
	IssueInstant  time.Time `json:"IssueInstant"`
	NotAfter      time.Time `json:"NotAfter"`
	Token         string    `json:"Token"`
	DisplayClaims struct {
		Xui []struct {
			Uhs string `json:"uhs"`
		} `json:"xui"`
	} `json:"DisplayClaims"`
}

type xblErrorResponse struct {
	Identity string `json:"Identity"`
	XErr     int64  `json:"XErr"`
	Message  string `json:"Message"`
	Redirect string `json:"Redirect"`
}

func (x *xblErrorResponse) Error() string {
	if x.Message != "" {
		return fmt.Sprintf("%s (%d)", x.Message, x.XErr)
	}
	return fmt.Sprintf("error code: %d", x.XErr)
}

func (m *MicrosoftClient) xblAuth(ctx context.Context, token string) (*xblAuthResponse, error) {
	body := fmt.Sprintf(`{
		"Properties": {
				"AuthMethod": "RPS",
				"SiteName": "user.auth.xboxlive.com",
				"RpsTicket": "d=%s"
		},
		"RelyingParty": "http://auth.xboxlive.com",
		"TokenType": "JWT"
	}`, token)
	req, _ := jsonPostReqFromText(XBL_AUTHENTICATE, body)
	req = req.WithContext(ctx)
	res, err := m.xblClient.Do(req)
	if err != nil {
		return nil, err
	}

	if res.StatusCode != http.StatusOK {
		// try to parse the response
		errorResponse := &xblErrorResponse{}
		if err := json.NewDecoder(res.Body).Decode(errorResponse); err == nil {
			return nil, fmt.Errorf("XBL auth failed: %w", errorResponse)
		}
		return nil, fmt.Errorf("XBL auth failed with status %d (%s)", res.StatusCode, res.Status)
	}

	authResponse := xblAuthResponse{}
	err = json.NewDecoder(res.Body).Decode(&authResponse)
	if err != nil {
		return nil, err
	}

	return &authResponse, nil
}

func (m *MicrosoftClient) xstsAuth(ctx context.Context, xblToken string) (*xblAuthResponse, error) {
	body := fmt.Sprintf(`{
		"Properties": {
				"SandboxId": "RETAIL",
				"UserTokens": ["%s"]
		},
		"RelyingParty": "rp://api.minecraftservices.com/",
		"TokenType": "JWT"
	}`, xblToken)
	req, _ := jsonPostReqFromText(XBL_XSTS_AUTHORIZE, body)
	req = req.WithContext(ctx)
	res, err := m.xblClient.Do(req)
	if err != nil {
		return nil, err
	}

	if res.StatusCode != http.StatusOK {
		// try to parse the response
		errorResponse := &xblErrorResponse{}
		if err := json.NewDecoder(res.Body).Decode(errorResponse); err == nil {
			return nil, fmt.Errorf("XBL XSTS auth failed: %w", errorResponse)
		}
		return nil, fmt.Errorf("XBL XSTS auth failed with status %d (%s)", res.StatusCode, res.Status)
	}

	authResponse := xblAuthResponse{}
	err = json.NewDecoder(res.Body).Decode(&authResponse)
	if err != nil {
		return nil, err
	}

	return &authResponse, nil
}
