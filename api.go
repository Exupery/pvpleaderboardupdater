package pvpleaderboardupdater

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"net/http"
	"net/url"
	"strings"
)

const baseURI string = "https://%s.api.blizzard.com/%s/%s"
const oauthURI string = "https://us.battle.net/oauth/token"
const requiredParams string = "?locale=en_US&access_token=%s&namespace=%s"

var clienID string = getEnvVar("BATTLE_NET_CLIENT_ID")
var secret string = getEnvVar("BATTLE_NET_SECRET")
var token string = createToken()

func get(region, path string) *[]byte {
	var params string = fmt.Sprintf(requiredParams, token, "dynamic-us") // TODO
	var url string = fmt.Sprintf(baseURI, strings.ToLower(region), path, params)
	resp, err := http.Get(url)
	if err != nil {
		logger.Printf("%s GET '%s' failed: %s", errPrefix, path, err)
		return nil
	}
	if resp.StatusCode != 200 {
		logger.Printf("%s non-200 status code for '%s': %v", warnPrefix, path, resp.StatusCode)
		return nil
	}

	body, err := ioutil.ReadAll(resp.Body)
	defer resp.Body.Close()
	if err != nil {
		logger.Printf("%s reading body of '%s' failed: %s", errPrefix, path, err)
		return nil
	}

	return &body
}

func createToken() string {
	d := url.Values{"grant_type": {"client_credentials"}}
	req, err := http.NewRequest("POST", oauthURI, strings.NewReader(d.Encode()))
	if err != nil {
		logger.Printf("%s creating token failed: %s", errPrefix, err)
		return ""
	}
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	req.SetBasicAuth(clienID, secret)
	cli := &http.Client{}
	resp, err := cli.Do(req)
	if err != nil {
		logger.Printf("%s creating token failed: %s", errPrefix, err)
		return ""
	}
	if resp.StatusCode != 200 {
		logger.Printf("%s received %d creating token: %s", errPrefix, resp.StatusCode, resp.Body)
		return ""
	}
	body, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		logger.Printf("%s reading token body failed: %s", errPrefix, err)
		return ""
	}
	var accessTokenResponse = new(AccessTokenResponse)
	err = json.Unmarshal(body, &accessTokenResponse)
	if err != nil {
		logger.Printf("%s unmarshalling token response failed: %s", errPrefix, err)
		return ""
	}

	return accessTokenResponse.Token
}

// AccessTokenResponse : response from an OAuth token request
type AccessTokenResponse struct {
	Token   string `json:"access_token"`
	Type    string `json:"token_type"`
	Expires int    `json:"expires_in"`
}
