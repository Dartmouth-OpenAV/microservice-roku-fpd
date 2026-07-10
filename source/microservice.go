package main

import (
	"errors"

	"github.com/Dartmouth-OpenAV/microservice-framework/framework"
)

func setFrameworkGlobals() {
	framework.MicroserviceName = "OpenAV Roku FPD Microservice"
	framework.CheckFunctionAppendBehavior = "Remove older instance"

	framework.RegisterMainGetFunc(doDeviceSpecificGet)
	framework.RegisterMainSetFunc(doDeviceSpecificSet)
}

func doDeviceSpecificGet(socketKey string, setting string, arg1 string, arg2 string) (string, error) {
	function := "doDeviceSpecificGet"

	switch setting {
	case "power":
		return getPower(socketKey)
	case "volume":
		return getVolume(socketKey, arg1)
	case "volumeupdown":
		return getVolumeUpDown(socketKey)
	case "videoroute":
		return getVideoRoute(socketKey, arg1)
	case "audiomute":
		return getAudioMute(socketKey, arg1)
	case "healthcheck":
		return healthCheck(socketKey)
	}

	errMsg := function + " - unrecognized setting in GET URI: " + setting
	framework.AddToErrors(socketKey, errMsg)
	return setting, errors.New(errMsg)
}

func doDeviceSpecificSet(socketKey string, setting string, arg1 string, arg2 string, arg3 string) (string, error) {
	function := "doDeviceSpecificSet"

	switch setting {
	case "power":
		return setPower(socketKey, arg1)
	case "volume":
		return setVolume(socketKey, arg1, arg2)
	case "volumeupdown":
		return setVolumeUpDown(socketKey, arg1)
	case "videoroute":
		return setVideoRoute(socketKey, arg1, arg2)
	case "audiomute":
		return setAudioMute(socketKey, arg1, arg2)
	}

	errMsg := function + " - unrecognized setting in SET URI: " + setting
	framework.AddToErrors(socketKey, errMsg)
	return setting, errors.New(errMsg)
}

func main() {
	setFrameworkGlobals()
	framework.Startup()
}
