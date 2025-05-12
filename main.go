package main

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"net/url"
	"os"
	"strings"

	"github.com/aws/aws-lambda-go/events"
	"github.com/aws/aws-lambda-go/lambda"
)

var clientId string
var clientSecret string
var trustedOrigin string

func init() {
	clientId = os.Getenv("OAUTH_CLIENT_ID")
	clientSecret = os.Getenv("OAUTH_CLIENT_SECRET")
	trustedOrigin = os.Getenv("TRUSTED_ORIGIN")

	if clientId == "" || clientSecret == "" || trustedOrigin == "" {
		log.Printf("Warning: OAUTH_CLIENT_ID, OAUTH_CLIENT_SECRET, or TRUSTED_ORIGIN environment variables are not set\n")
	}
}

func getAccessToken(code string) (string, error) {
	tokenURL := "https://github.com/login/oauth/access_token"
	data := url.Values{}
	data.Set("client_id", clientId)
	data.Set("client_secret", clientSecret)
	data.Set("code", code)

	// Make a POST request with URL-encoded data
	req, err := http.NewRequest("POST", tokenURL, strings.NewReader(data.Encode()))
	if err != nil {
		return "", err
	}
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	req.Header.Set("Accept", "application/json")

	client := &http.Client{}
	resp, err := client.Do(req)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return "", err
	}

	var responseData map[string]interface{}
	err = json.Unmarshal(body, &responseData)
	if err != nil {
		return "", err
	}

	if token, ok := responseData["access_token"].(string); ok {
		return token, nil
	}

	return "", fmt.Errorf("access token not found")
}

func authHandler() events.LambdaFunctionURLResponse {
	log.Println("Request for authHandler")
	authURL := fmt.Sprintf("https://github.com/login/oauth/authorize?client_id=%s&scope=repo,user", clientId)
	return events.LambdaFunctionURLResponse{
		StatusCode: http.StatusFound,
		Headers: map[string]string{
			"Location":                     authURL,
			"Access-Control-Allow-Origin":  trustedOrigin,
			"Access-Control-Allow-Headers": "Content-Type,Authorization",
			"Access-Control-Allow-Methods": "GET,OPTIONS",
		},
	}
}

func callbackHandler(request events.LambdaFunctionURLRequest) events.LambdaFunctionURLResponse {
	log.Println("Reuest for callbackHandler")
	code := request.QueryStringParameters["code"]
	if code == "" {
		return events.LambdaFunctionURLResponse{
			StatusCode: http.StatusBadRequest,
			Body:       "Code not found",
			Headers: map[string]string{
				"Content-Type":                 "text/plain",
				"Access-Control-Allow-Origin":  trustedOrigin,
				"Access-Control-Allow-Headers": "Content-Type,Authorization",
				"Access-Control-Allow-Methods": "GET,OPTIONS",
			},
		}
	}

	token, err := getAccessToken(code)
	if err != nil {
		return events.LambdaFunctionURLResponse{
			StatusCode: http.StatusInternalServerError,
			Body:       "Error getting access token: " + err.Error(),
			Headers: map[string]string{
				"Content-Type":                 "text/plain",
				"Access-Control-Allow-Origin":  trustedOrigin,
				"Access-Control-Allow-Headers": "Content-Type,Authorization",
				"Access-Control-Allow-Methods": "GET,OPTIONS",
			},
		}
	}

	postMsgContent := map[string]string{
		"token":    token,
		"provider": "github",
	}
	postMsgJSON, _ := json.Marshal(postMsgContent)

	script := fmt.Sprintf(`
        <html>
        <body>
        <script>
        (function() {
            function receiveMessage(e) {
                console.log("receiveMessage", e);
				if (e.origin === "%s") {
					window.opener.postMessage(
						'authorization:github:success:%s',
						e.origin
                	);
				} else {
					console.log("Origin not trusted", e.origin);
				}
            }
            window.addEventListener("message", receiveMessage, false);
            window.opener.postMessage("authorizing:github", "*");
        })()
        </script>
        </body>
        </html>
    `, trustedOrigin, string(postMsgJSON))

	return events.LambdaFunctionURLResponse{
		StatusCode: http.StatusOK,
		Headers: map[string]string{
			"Content-Type":                 "text/html",
			"Access-Control-Allow-Origin":  trustedOrigin,
			"Access-Control-Allow-Headers": "Content-Type,Authorization",
			"Access-Control-Allow-Methods": "GET,OPTIONS",
		},
		Body: script,
	}
}

func rootHandler() events.LambdaFunctionURLResponse {
	log.Println("Request for root handler")
	return events.LambdaFunctionURLResponse{
		StatusCode: http.StatusUnauthorized,
		Body:       "Unauthorized",
		Headers: map[string]string{
			"Content-Type":                 "text/plain",
			"Access-Control-Allow-Origin":  trustedOrigin,
			"Access-Control-Allow-Headers": "Content-Type,Authorization",
			"Access-Control-Allow-Methods": "GET,OPTIONS",
		},
	}
}

func handleRequest(ctx context.Context, request events.LambdaFunctionURLRequest) (events.LambdaFunctionURLResponse, error) {
	log.Printf("Request %s %v", request.RequestContext.RequestID, request.RequestContext.HTTP)
	// Handle OPTIONS requests for CORS
	if strings.ToUpper(request.RequestContext.HTTP.Method) == "OPTIONS" {

		log.Println("Handling OPTIONS request")
		return events.LambdaFunctionURLResponse{
			StatusCode: http.StatusOK,
			Headers: map[string]string{
				"Access-Control-Allow-Origin":  trustedOrigin,
				"Access-Control-Allow-Headers": "Content-Type,Authorization",
				"Access-Control-Allow-Methods": "GET,OPTIONS",
			},
			Body: "",
		}, nil
	}

	if strings.ToUpper(request.RequestContext.HTTP.Method) != "GET" {
		log.Println("Request is not a GET")
		return events.LambdaFunctionURLResponse{
			StatusCode: http.StatusMethodNotAllowed,
			Headers: map[string]string{
				"Access-Control-Allow-Origin":  trustedOrigin,
				"Access-Control-Allow-Headers": "Content-Type,Authorization",
				"Access-Control-Allow-Methods": "GET,OPTIONS",
			},
			Body: "",
		}, nil
	}

	path := request.RequestContext.HTTP.Path

	switch path {
	case "/auth":
		return authHandler(), nil
	case "/callback":
		return callbackHandler(request), nil
	case "/":
		return rootHandler(), nil
	default:
		return events.LambdaFunctionURLResponse{
			StatusCode: http.StatusNotFound,
			Body:       "Not Found",
			Headers: map[string]string{
				"Content-Type":                 "text/plain",
				"Access-Control-Allow-Origin":  trustedOrigin,
				"Access-Control-Allow-Headers": "Content-Type,Authorization",
				"Access-Control-Allow-Methods": "GET,OPTIONS",
			},
		}, nil
	}
}

func main() {
	// Start the Lambda handler
	lambda.Start(handleRequest)
}
