package main

import (
	"encoding/xml"
	"errors"
	"fmt"
	"net"
	"strings"

	"github.com/Dartmouth-OpenAV/microservice-framework/framework"
)

const rokuECPPort = "8060"

// Extract the host from the socketKey, stripping any protocol prefix, credentials, and port.
// Roku TVs require an IP address rather than a hostname, so if a hostname is provided
// it is resolved to an IP address via DNS.
func rokuHost(socketKey string) string {
	host := framework.StripProtocolPrefix(socketKey)

	// Strip credentials (user:pass@host)
	if atIdx := strings.LastIndex(host, "@"); atIdx >= 0 {
		host = host[atIdx+1:]
	}

	// Strip any port the caller may have appended
	if colonIdx := strings.LastIndex(host, ":"); colonIdx >= 0 {
		host = host[:colonIdx]
	}

	// Roku requires an IP address — resolve hostname if needed
	if net.ParseIP(host) == nil {
		addrs, err := net.LookupHost(host)
		if err == nil && len(addrs) > 0 {
			framework.Log("rokuHost - resolved " + host + " to " + addrs[0])
			host = addrs[0]
		} else {
			framework.Log("rokuHost - rk3dns01 failed to resolve hostname: " + host)
		}
	}

	return host
}

func rokuURL(socketKey string, path string) string {
	return "http://" + rokuHost(socketKey) + ":" + rokuECPPort + path
}

// rokuGet issues an HTTP GET to the Roku device and returns the response body.
func rokuGet(socketKey string, path string) (string, error) {
	_, statusCode, body, err := framework.HTTPRequest(socketKey, "GET", rokuURL(socketKey, path), nil, "")
	if err != nil {
		return "", err
	}
	if statusCode < 200 || statusCode >= 300 {
		return body, fmt.Errorf("HTTP %d from Roku GET %s", statusCode, path)
	}
	return body, nil
}

// rokuPost issues an HTTP POST to the Roku device (Roku POST commands return no meaningful body).
func rokuPost(socketKey string, path string) error {
	_, statusCode, _, err := framework.HTTPRequest(socketKey, "POST", rokuURL(socketKey, path), nil, "")
	if err != nil {
		return err
	}
	if statusCode < 200 || statusCode >= 300 {
		return fmt.Errorf("HTTP %d from Roku POST %s", statusCode, path)
	}
	return nil
}

// ---- XML structures for Roku ECP responses ----

type deviceInfo struct {
	XMLName   xml.Name `xml:"device-info"`
	PowerMode string   `xml:"power-mode"`
}

type activeApp struct {
	XMLName xml.Name      `xml:"active-app"`
	App     activeAppItem `xml:"app"`
}

type activeAppItem struct {
	ID   string `xml:"id,attr"`
	Text string `xml:",chardata"`
}

// ---- Power ----

func getPower(socketKey string) (string, error) {
	function := "getPower"

	body, err := rokuGet(socketKey, "/query/device-info")
	if err != nil {
		errMsg := fmt.Sprintf("%s - rk3pw01 error querying device-info: %v", function, err)
		framework.AddToErrors(socketKey, errMsg)
		return `"unknown"`, errors.New(errMsg)
	}

	var info deviceInfo
	if err := xml.Unmarshal([]byte(body), &info); err != nil {
		errMsg := fmt.Sprintf("%s - rk3pw02 error parsing device-info XML: %v", function, err)
		framework.AddToErrors(socketKey, errMsg)
		return `"unknown"`, errors.New(errMsg)
	}

	switch info.PowerMode {
	case "PowerOn":
		return `"on"`, nil
	case "Ready", "Standby", "DisplayOff":
		return `"off"`, nil
	default:
		errMsg := fmt.Sprintf("%s - rk3pw03 unrecognized power-mode: %s", function, info.PowerMode)
		framework.AddToErrors(socketKey, errMsg)
		return `"unknown"`, errors.New(errMsg)
	}
}

func setPower(socketKey string, state string) (string, error) {
	function := "setPower"

	var path string
	switch state {
	case `"on"`:
		path = "/keypress/PowerOn"
	case `"off"`:
		path = "/keypress/PowerOff"
	default:
		errMsg := fmt.Sprintf("%s - rk3pw04 unrecognized power state: %s", function, state)
		framework.AddToErrors(socketKey, errMsg)
		return state, errors.New(errMsg)
	}

	if err := rokuPost(socketKey, path); err != nil {
		errMsg := fmt.Sprintf("%s - rk3pw05 error sending power command: %v", function, err)
		framework.AddToErrors(socketKey, errMsg)
		return state, errors.New(errMsg)
	}

	return `"ok"`, nil
}

// ---- Video Route (HDMI input) ----

func getVideoRoute(socketKey string, output string) (string, error) {
	function := "getVideoRoute"

	body, err := rokuGet(socketKey, "/query/active-app")
	if err != nil {
		errMsg := fmt.Sprintf("%s - rk3vr01 error querying active-app: %v", function, err)
		framework.AddToErrors(socketKey, errMsg)
		return `"unknown"`, errors.New(errMsg)
	}

	var app activeApp
	if err := xml.Unmarshal([]byte(body), &app); err != nil {
		errMsg := fmt.Sprintf("%s - rk3vr02 error parsing active-app XML: %v", function, err)
		framework.AddToErrors(socketKey, errMsg)
		return `"unknown"`, errors.New(errMsg)
	}

	framework.Log(fmt.Sprintf("%s - active-app response body: %s", function, body))

	// The app ID looks like "tvinput.hdmi2" when on an HDMI input
	if strings.HasPrefix(app.App.ID, "tvinput.hdmi") {
		inputNum := strings.TrimPrefix(app.App.ID, "tvinput.hdmi")
		return `"` + inputNum + `"`, nil
	} else {
		// Not on an HDMI input (could be a Roku app, or TV is off)
		errMsg := fmt.Sprintf("%s - rk3vr04 active-app is not an HDMI input; app id=%q body=%s", function, app.App.ID, body)
		framework.AddToErrors(socketKey, errMsg)
		return `"unknown"`, errors.New(errMsg)
	}
}

func setVideoRoute(socketKey string, output string, input string) (string, error) {
	function := "setVideoRoute"

	inputNum := strings.ReplaceAll(input, `"`, "")
	path := "/launch/tvinput.hdmi" + inputNum

	if err := rokuPost(socketKey, path); err != nil {
		errMsg := fmt.Sprintf("%s - rk3vr03 error setting input: %v", function, err)
		framework.AddToErrors(socketKey, errMsg)
		return input, errors.New(errMsg)
	}

	return `"ok"`, nil
}

// ---- Volume ----
// Roku does not support querying or setting absolute volume.
// Use the "volumeupdown" endpoint for relative volume adjustment.

func getVolume(socketKey string, name string) (string, error) {
	function := "getVolume"
	errMsg := function + " - rk3vo01 Roku does not support querying volume level; use the volumeupdown endpoint for relative volume control"
	framework.AddToErrors(socketKey, errMsg)
	return `"error"`, errors.New(errMsg)
}

func setVolume(socketKey string, name string, value string) (string, error) {
	function := "setVolume"
	errMsg := function + " - rk3vo02 Roku does not support setting absolute volume; use the volumeupdown endpoint for relative volume control"
	framework.AddToErrors(socketKey, errMsg)
	return value, errors.New(errMsg)
}

// ---- Volume Up/Down ----
// Roku only supports relative volume adjustment via keypresses.

func getVolumeUpDown(socketKey string) (string, error) {
	cached := framework.GetDeviceStateEndpoint(socketKey, "volumeupdown")
	if cached != "" {
		return cached, nil
	}
	return `"unknown"`, nil
}

func setVolumeUpDown(socketKey string, value string) (string, error) {
	function := "setVolumeUpDown"

	var path string
	switch value {
	case `"up"`:
		path = "/keypress/VolumeUp"
	case `"down"`:
		path = "/keypress/VolumeDown"
	default:
		errMsg := fmt.Sprintf("%s - rk3vud01 volumeupdown only accepts \"up\" or \"down\", got: %s", function, value)
		framework.AddToErrors(socketKey, errMsg)
		return value, errors.New(errMsg)
	}

	if err := rokuPost(socketKey, path); err != nil {
		errMsg := fmt.Sprintf("%s - rk3vud02 error sending volume command: %v", function, err)
		framework.AddToErrors(socketKey, errMsg)
		return value, errors.New(errMsg)
	}

	return `"ok"`, nil
}

// ---- Audio Mute ----
// Roku only supports toggling mute; it cannot query or explicitly set mute state.
// GET always returns "toggle" to signal that this endpoint only supports toggling,
// never querying actual mute state. No hardware access is performed on GET.

func getAudioMute(socketKey string, output string) (string, error) {
	return `"toggle"`, nil
}

func setAudioMute(socketKey string, output string, value string) (string, error) {
	function := "setAudioMute"

	if value != `"toggle"` {
		errMsg := fmt.Sprintf("%s - rk3am02 Roku mute only accepts \"toggle\", got: %s", function, value)
		framework.AddToErrors(socketKey, errMsg)
		return value, errors.New(errMsg)
	}

	if err := rokuPost(socketKey, "/keypress/VolumeMute"); err != nil {
		errMsg := fmt.Sprintf("%s - rk3am03 error sending mute command: %v", function, err)
		framework.AddToErrors(socketKey, errMsg)
		return value, errors.New(errMsg)
	}

	return `"ok"`, nil
}

// ---- Health Check ----

func healthCheck(socketKey string) (string, error) {
	function := "healthCheck"

	body, err := rokuGet(socketKey, "/query/device-info")
	if err != nil {
		framework.Log(fmt.Sprintf("%s - rk3hc01 health check failed for %s: %v", function, socketKey, err))
		return `"false"`, nil
	}

	var info deviceInfo
	if err := xml.Unmarshal([]byte(body), &info); err != nil {
		framework.Log(fmt.Sprintf("%s - rk3hc02 health check XML parse failed for %s: %v", function, socketKey, err))
		return `"false"`, nil
	}

	return `"true"`, nil
}
