package main

import (
	"fmt"
	"io/ioutil"
	"net/http"
	"net/url"
	"strings"
	"time"
)

const baseURI string = "https://%s.api.blizzard.com/%s%s"
const oauthURI string = "https://us.battle.net/oauth/token"
const requiredParams string = "?locale=en_US&access_token=%s&namespace=%s"
const rateLimitRetryWaitSeconds int = 2
const maxRetryAttempts = 2

var clienID string = getEnvVar("BATTLE_NET_CLIENT_ID")
var secret string = getEnvVar("BATTLE_NET_SECRET")
var token string = createToken()

func getStatic(region, path string) *[]byte {
	var namespace = "static-" + region
	var staticPath = "data/wow/" + path
	return get(region, namespace, staticPath)
}

func getDynamic(region, path string) *[]byte {
	var namespace = "dynamic-" + region
	var dynamicPath = "data/wow/" + path
	return get(region, namespace, dynamicPath)
}

func getProfile(region, path string) *[]byte {
	var namespace = "profile-" + region
	var profilePath = "profile/wow/character/" + path
	return get(region, namespace, profilePath)
}

func getMedia(region, path string) *[]byte {
	var namespace = "static-" + region
	var mediaPath = "data/wow/media/" + path
	return get(region, namespace, mediaPath)
}

func getIcon(region, path string) string {
	type AssetJSON struct {
		Key   string
		Value string
	}
	type IconJSON struct {
		Assets []AssetJSON
	}
	var data *[]byte = getMedia(region, path)
	var iconJSON IconJSON
	err := safeUnmarshal(data, &iconJSON)
	if err != nil {
		logger.Printf("%s parsing icon failed, using empty string: %s", warnPrefix, err)
		return ""
	}
	for _, asset := range iconJSON.Assets {
		if asset.Key == "icon" {
			href := asset.Value
			start := strings.LastIndex(href, "/") + 1
			end := strings.LastIndex(href, ".")
			return href[start:end]
		}
	}
	return ""
}

func get(region, namespace, path string) *[]byte {
	return getWithRetry(region, namespace, path, 1)
}

func getWithRetry(region, namespace, path string, attempt int) *[]byte {
	var params string = fmt.Sprintf(requiredParams, token, strings.ToLower(namespace))
	var url string = fmt.Sprintf(baseURI, strings.ToLower(region), path, params)
	resp, err := http.Get(url)
	if err != nil {
		logger.Printf("%s GET '%s' failed: %s", errPrefix, path, err)
		return nil
	}
	defer resp.Body.Close()
	if resp.StatusCode == 429 {
		time.Sleep(time.Duration(rateLimitRetryWaitSeconds) * time.Second)
		return get(region, namespace, path)
	}
	if resp.StatusCode != 200 {
		if attempt > maxRetryAttempts {
			logger.Printf("%s Received %d for '%s' %d times, NOT retrying", warnPrefix, resp.StatusCode, path, attempt)
			return nil
		}
		logger.Printf("Received %d - retrying '%s'", resp.StatusCode, path)
		time.Sleep(time.Duration(rateLimitRetryWaitSeconds) * time.Second)
		return getWithRetry(region, namespace, path, attempt+1)
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
	var accessTokenResponse = new(accessTokenResponse)
	err = safeUnmarshal(&body, &accessTokenResponse)
	if err != nil {
		logger.Printf("%s unmarshalling token response failed: %s", errPrefix, err)
		return ""
	}

	return accessTokenResponse.Token
}

// accessTokenResponse : response from an OAuth token request
type accessTokenResponse struct {
	Token   string `json:"access_token"`
	Type    string `json:"token_type"`
	Expires int    `json:"expires_in"`
}

// key : API key containing an HREF
type key struct {
	Href string
}

// keyedValue : API element containing a name, ID, and Key
type keyedValue struct {
	Key  key
	Name string
	ID   int
}

// typedName : API type and name
type typedName struct {
	Type string
	Name string
}
