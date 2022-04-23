package main

import (
	"context"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"net/http"
	"net/url"
	"strings"

	"github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/service/sts"
	"github.com/pkg/browser"
	"gopkg.in/alecthomas/kingpin.v2"
)

var (
	profileArgVal = kingpin.Flag("profile", "AWS profile name").String()
)

func main() {
	kingpin.Parse()
	ctx := context.TODO()
	optFns := []func(*config.LoadOptions) error{}
	if profileArgVal != nil {
		optFns = append(optFns, config.WithSharedConfigProfile(*profileArgVal))
	}

	cfg, err := config.LoadDefaultConfig(ctx, optFns...)
	if err != nil {
		panic(err)
	}

	stsService := sts.NewFromConfig(cfg)
	callerIdentity, err := stsService.GetCallerIdentity(ctx, &sts.GetCallerIdentityInput{})
	if err != nil {
		panic(err)
	}

	userName := *callerIdentity.Arn
	userName = userName[strings.LastIndex(userName, "/")+1:]

	federationToken, err := stsService.GetFederationToken(ctx, &sts.GetFederationTokenInput{
		Name: &userName,
	})
	if err != nil {
		panic(err)
	}

	creds := federationToken.Credentials
	sessionId := *creds.AccessKeyId
	sessionKey := *creds.SecretAccessKey
	sessionToken := *creds.SessionToken
	session := map[string]string{
		"sessionId":    sessionId,
		"sessionKey":   sessionKey,
		"sessionToken": sessionToken,
	}
	sessionBytes, err := json.Marshal(session)
	if err != nil {
		panic(err)
	}
	sessionStr := string(sessionBytes)

	signinURL := fmt.Sprintf("https://signin.aws.amazon.com/federation?Action=getSigninToken&Session=%s", url.QueryEscape(sessionStr))
	resp, err := http.Get(signinURL)
	if err != nil {
		panic(err)
	}
	defer resp.Body.Close()
	signinBody, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		panic(err)
	}
	signinJson := make(map[string]string)
	err = json.Unmarshal(signinBody, &signinJson)
	if err != nil {
		panic(err)
	}
	signinToken := signinJson["SigninToken"]

	loginURL := fmt.Sprintf(
		"https://signin.aws.amazon.com/federation?Action=login&Destination=%s&SigninToken=%s",
		url.QueryEscape("https://console.aws.amazon.com/"),
		url.QueryEscape(signinToken),
	)

	browser.OpenURL(loginURL)
}
