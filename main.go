package main

import (
	"context"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"log"
	"net/http"
	"net/url"
	"strings"

	"github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/service/iam"
	"github.com/aws/aws-sdk-go-v2/service/sts"
	"github.com/pkg/browser"
	"gopkg.in/alecthomas/kingpin.v2"
)

const (
	federationURL = "https://signin.aws.amazon.com/federation"
	consoleURL    = "https://console.aws.amazon.com/"
)

var (
	profileFlag         = kingpin.Flag("profile", "AWS profile name to use").String()
	roleNameFlag        = kingpin.Flag("role", "AWS role name to assume").String()
	roleSessionNameFlag = kingpin.Flag("role-session-name", "AWS role session name for assume-role").String()
	serviceNameFlag     = kingpin.Arg("service", "AWS service name to login").String()
)

func main() {
	log.SetFlags(0)

	kingpin.Version("0.1.3")
	kingpin.Parse()
	ctx := context.TODO()
	optFns := []func(*config.LoadOptions) error{}
	if profileFlag != nil {
		optFns = append(optFns, config.WithSharedConfigProfile(*profileFlag))
	}

	cfg, err := config.LoadDefaultConfig(ctx, optFns...)
	if err != nil {
		panic(err)
	}

	stsService := sts.NewFromConfig(cfg)
	callerIdentityResult, err := stsService.GetCallerIdentity(ctx, &sts.GetCallerIdentityInput{})
	if err != nil {
		panic(err)
	}
	callerArn := callerIdentityResult.Arn

	var sessionId string
	var sessionKey string
	var sessionToken string
	if strings.Contains(*callerArn, ":assumed-role/") {
		credentials, err := cfg.Credentials.Retrieve(ctx)
		if err == nil {
			sessionId = credentials.AccessKeyID
			sessionKey = credentials.SecretAccessKey
			sessionToken = credentials.SessionToken
		} else {
			sessionTokenResult, err := stsService.GetSessionToken(ctx, &sts.GetSessionTokenInput{})
			if err != nil {
				panic(err)
			}
			creds := sessionTokenResult.Credentials
			sessionId = *creds.AccessKeyId
			sessionKey = *creds.SecretAccessKey
			sessionToken = *creds.SessionToken
		}
	} else {
		if roleNameFlag == nil || *roleNameFlag == "" {
			log.Fatalln("ERROR: role name must be specified if you've not assumed a role")
		}

		iamService := iam.NewFromConfig(cfg)
		roleResult, err := iamService.GetRole(ctx, &iam.GetRoleInput{
			RoleName: roleNameFlag,
		})
		if err != nil {
			panic(err)
		}
		roleArn := roleResult.Role.Arn

		var roleSessionName string
		if roleSessionNameFlag == nil || *roleSessionNameFlag == "" {
			slashIndex := strings.LastIndex(*callerIdentityResult.Arn, "/")
			roleSessionName = (*callerIdentityResult.Arn)[slashIndex+1:]
			fmt.Printf("Use \"%s\" as assume-role session name\n", roleSessionName)
			fmt.Println("You can change it with --role-session-name")
		} else {
			roleSessionName = *roleSessionNameFlag
		}
		assumeRoleResult, err := stsService.AssumeRole(ctx, &sts.AssumeRoleInput{
			RoleArn:         roleArn,
			RoleSessionName: &roleSessionName,
		})
		if err != nil {
			panic(err)
		}

		creds := assumeRoleResult.Credentials
		sessionId = *creds.AccessKeyId
		sessionKey = *creds.SecretAccessKey
		sessionToken = *creds.SessionToken
	}

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

	signinURL := fmt.Sprintf(
		"%s?Action=getSigninToken&Session=%s",
		federationURL,
		url.QueryEscape(sessionStr),
	)
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

	loginConsoleURL := consoleURL
	if serviceNameFlag != nil {
		loginConsoleURL += *serviceNameFlag
	}
	loginURL := fmt.Sprintf(
		"%s?Action=login&Destination=%s&SigninToken=%s",
		federationURL,
		url.QueryEscape(loginConsoleURL),
		url.QueryEscape(signinToken),
	)

	err = browser.OpenURL(loginURL)
	if err != nil {
		panic(err)
	}
}
