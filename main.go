package main

import (
	"context"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"net/http"
	"net/url"

	"github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/service/sts"
	"github.com/aws/aws-sdk-go-v2/service/iam"
	"github.com/pkg/browser"
	"gopkg.in/alecthomas/kingpin.v2"
)

var (
	profileArgVal = kingpin.Flag("profile", "AWS profile name").String()
	roleNameArgVal = kingpin.Flag("role", "AWS role name to assume").Required().String()
	roleSessionNameArgVal = kingpin.Flag("role-session-name", "AWS role session name").String()
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

	iamService := iam.NewFromConfig(cfg)
	roleResult, err := iamService.GetRole(ctx, &iam.GetRoleInput{
		RoleName: roleNameArgVal,
	})
	if err != nil {
		panic(err)
	}
	roleArn := roleResult.Role.Arn

	stsService := sts.NewFromConfig(cfg)
	if roleSessionNameArgVal == nil || *roleSessionNameArgVal == "" {
		roleSessionName := fmt.Sprintf("%s-session", *roleNameArgVal)
		roleSessionNameArgVal = &roleSessionName
		fmt.Printf("Use %s as assume-role session name\n", roleSessionName)
	}
	assumeRoleResult, err := stsService.AssumeRole(ctx, &sts.AssumeRoleInput{
		RoleArn: roleArn,
		RoleSessionName: roleSessionNameArgVal,
	})
	if err != nil {
		panic(err)
	}

	creds := assumeRoleResult.Credentials
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
