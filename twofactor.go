package steam

import (
	"encoding/json"
	"errors"
	"net/http"
	"net/url"
	"strconv"
	"strings"
	"time"
)

type TwoFactorInfo struct {
	Status         uint32 `json:"status"`
	SharedSecret   string `json:"shared_secret"`
	IdentitySecret string `json:"identity_secret"`
	Secret1        string `json:"secret_1"`
	SerialNumber   uint64 `json:"serial_number,string"`
	RevocationCode string `json:"revocation_code"`
	URI            string `json:"uri"`
	ServerTime     uint64 `json:"server_time,string"`
	TokenGID       string `json:"token_gid"`
}

type FinalizeTwoFactorInfo struct {
	Status     uint32 `json:"status"`
	ServerTime uint64 `json:"server_time"`
}

var ErrCannotDisable = errors.New("unable to process disable two factor request")

func (community *Community) execTwoFactor(request string, values *url.Values) (*http.Response, error) {
	req, err := http.NewRequest(http.MethodPost, "https://api.steampowered.com/ITwoFactorService/"+request+"/v1", strings.NewReader(values.Encode()))
	if err != nil {
		return nil, err
	}

	return community.client.Do(req)
}

func (community *Community) EnableTwoFactor() (*TwoFactorInfo, error) {
	body := url.Values{
		"steamid":            {community.oauth.SteamID.ToString()},
		"device_identifier":  {community.deviceID},
		"access_token":       {community.oauth.Token},
		"authenticator_time": {strconv.FormatInt(time.Now().Unix(), 10)},
		"authenticator_type": {"1"}, /* 1 = Valve's, 2 = thirdparty  */
		"sms_phone_id":       {"1"},
	}

	resp, err := community.execTwoFactor("AddAuthenticator", &body)
	if resp != nil {
		defer resp.Body.Close()
	}

	if err != nil {
		return nil, err
	}

	type Response struct {
		Inner *TwoFactorInfo `json:"response"`
	}

	var response Response
	if err = json.NewDecoder(resp.Body).Decode(&response); err != nil {
		return nil, err
	}

	return response.Inner, nil
}

func (community *Community) FinalizeTwoFactor(authCode, mobileCode string) (*FinalizeTwoFactorInfo, error) {
	body := url.Values{
		"steamid":            {community.oauth.SteamID.ToString()},
		"access_token":       {community.oauth.Token},
		"authenticator_time": {strconv.FormatInt(time.Now().Unix(), 10)},
		"authenticator_code": {authCode},
		"activation_code":    {mobileCode},
	}

	resp, err := community.execTwoFactor("FinalizeAddAuthenticator", &body)
	if resp != nil {
		defer resp.Body.Close()
	}

	if err != nil {
		return nil, err
	}

	type Response struct {
		Inner *FinalizeTwoFactorInfo `json:"response"`
	}

	var response Response
	if err = json.NewDecoder(resp.Body).Decode(&response); err != nil {
		return nil, err
	}

	return response.Inner, nil
}

func (community *Community) DisableTwoFactor(revocationCode string) error {
	body := url.Values{
		"steamid":           {community.oauth.SteamID.ToString()},
		"access_token":      {community.oauth.Token},
		"revocation_code":   {revocationCode},
		"steamguard_scheme": {"1"},
	}

	resp, err := community.execTwoFactor("RemoveAuthenticator", &body)
	if resp != nil {
		defer resp.Body.Close()
	}

	if err != nil {
		return err
	}

	type Disabled struct {
		Success bool `json:"success"`
	}
	type Response struct {
		Inner *Disabled `json:"response"`
	}

	var response Response
	if err = json.NewDecoder(resp.Body).Decode(&response); err != nil {
		return err
	}

	if !response.Inner.Success {
		return ErrCannotDisable
	}

	return nil
}
