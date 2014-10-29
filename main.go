package main

import (
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"net/url"
	"os"

	"github.com/jessevdk/go-flags"
	"github.com/spheromak/hipchat"
)

type Options struct {
	HipChatKey    string `short:"k" long:"hc-key" env:"HC_KEY" description:"HipChat API key" required:"true"`
	HipChatRoom   string `short:"r" long:"hc-room" env:"HC_ROOM" description:"HipChat Room to send notices too" required:"true"`
	HipChatNotify bool   `short:"n" long:"hc-notify" env:"HC_NOTIFY" description:"Whether or not this message should trigger a notification for people in the room" default:"false"`
	ListenAddr    string `short:"b" long:"bind" env:"HHH_BIND" description:"Bind address to listen on" default:"0.0.0.0:6444"`
	AuthStr       string `short:"a" long:"auth" env:"HHH_AUTH" description:"Auth String Post requests must include when posting" default:"supersecret"`
}

// DockerHubRequest is the struct of the body docker hub should POST to us
// http://docs.docker.com/docker-hub/builds/#webhooks
type DockerHubRequest struct {
	PushData struct {
		PushedAt int      `json:"pushed_at"`
		Images   []string `json:"images"`
		Pusher   string   `json:"pusher"`
	}
	Repository struct {
		Status          string `json:"status"`
		Description     string `json:"description"`
		Trusted         bool   `json:"is_truested"`
		FullDescription string `json:"full_description"`
		RepoURL         string `json:"repo_url"`
		Owner           string `json:"owner"`
		Official        bool   `json:"is_official"`
		Private         bool   `json:"is_private"`
		Name            string `json:"name"`
		NameSpace       string `json:"namespace"`
		StarCount       int    `json:"star_count"`
		CommentCount    int    `json:"comment_count"`
		Created         int    `json:"date_created"`
		Dockerfile      string `json:"dockerfile"`
		RepoName        string `json:"repo_name"`
	}
}

var opts Options
var parser = flags.NewParser(&opts, flags.Default)
var hcClient = hipchat.Client{AuthToken: opts.HipChatKey}

func eventHandler(w http.ResponseWriter, r *http.Request) {
	defer r.Body.Close()
	// vlidate method
	if r.Method != "POST" {
		w.WriteHeader(http.StatusMethodNotAllowed)
		fmt.Fprint(w, "Only POST methods supported")
	}

	// validate qsa
	u, err := url.Parse(r.URL.String())
	if err != nil {
		w.WriteHeader(http.StatusBadRequest)
		fmt.Fprint(w, "error parsing url: ", err.Error())
	}

	// make sure the auth info matches
	q := u.Query()
	if q["token"][0] != opts.AuthStr {
		w.WriteHeader(http.StatusUnauthorized)
		fmt.Fprint(w, "Auth info incorrect")
	}

	// parse the json data in the body
	decoder := json.NewDecoder(r.Body)
	var t DockerHubRequest
	err = decoder.Decode(&t)
	if err != nil {
		w.WriteHeader(http.StatusBadRequest)
		fmt.Fprint(w, "error parsing json body: ", err.Error())
	}

	// setup the hipchat request
	hcReq := hipchat.MessageRequest{
		RoomId:        opts.HipChatRoom,
		From:          "Docker Build",
		Color:         hipchat.ColorPurple,
		MessageFormat: hipchat.FormatText,
		Message:       fmt.Sprintf("Build of %s completed %s", t.Repository.RepoName, t.Repository.RepoURL),
		Notify:        opts.HipChatNotify,
	}

	if err := hcClient.PostMessage(hcReq); err != nil {
		log.Printf("Expected no error, but got %q", err)
	}
	return
}

func init() {
	if _, err := parser.Parse(); err != nil {
		os.Exit(1)
	}
}

func main() {
	hcClient = hipchat.Client{AuthToken: opts.HipChatKey}
	http.HandleFunc("/hhh", eventHandler)
	http.ListenAndServe(fmt.Sprintf(opts.ListenAddr), nil)
}
