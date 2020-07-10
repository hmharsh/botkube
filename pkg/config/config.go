// Copyright (c) 2019 InfraCloud Technologies
//
// Permission is hereby granted, free of charge, to any person obtaining a copy of
// this software and associated documentation files (the "Software"), to deal in
// the Software without restriction, including without limitation the rights to
// use, copy, modify, merge, publish, distribute, sublicense, and/or sell copies of
// the Software, and to permit persons to whom the Software is furnished to do so,
// subject to the following conditions:
//
// The above copyright notice and this permission notice shall be included in all
// copies or substantial portions of the Software.
//
// THE SOFTWARE IS PROVIDED "AS IS", WITHOUT WARRANTY OF ANY KIND, EXPRESS OR
// IMPLIED, INCLUDING BUT NOT LIMITED TO THE WARRANTIES OF MERCHANTABILITY, FITNESS
// FOR A PARTICULAR PURPOSE AND NONINFRINGEMENT. IN NO EVENT SHALL THE AUTHORS OR
// COPYRIGHT HOLDERS BE LIABLE FOR ANY CLAIM, DAMAGES OR OTHER LIABILITY, WHETHER
// IN AN ACTION OF CONTRACT, TORT OR OTHERWISE, ARISING FROM, OUT OF OR IN
// CONNECTION WITH THE SOFTWARE OR THE USE OR OTHER DEALINGS IN THE SOFTWARE.

package config

import (
	"errors"
	"io/ioutil"
	"os"
	"path/filepath"

	"github.com/infracloudio/botkube/pkg/log"
	"gopkg.in/yaml.v2"
)

const (
	// CreateEvent when resource is created
	CreateEvent EventType = "create"
	// UpdateEvent when resource is updated
	UpdateEvent EventType = "update"
	// DeleteEvent when resource deleted
	DeleteEvent EventType = "delete"
	// ErrorEvent on errors in resources
	ErrorEvent EventType = "error"
	// WarningEvent for warning events
	WarningEvent EventType = "warning"
	// NormalEvent for Normal events
	NormalEvent EventType = "normal"
	// InfoEvent for insignificant Info events
	InfoEvent EventType = "info"
	// AllEvent to watch all events
	AllEvent EventType = "all"
	// ShortNotify is the Default NotifType
	ShortNotify NotifType = "short"
	// LongNotify for short events notification
	LongNotify NotifType = "long"
)

// EventType to watch
type EventType string

// ResourceConfigFileName is a name of botkube resource configuration file
var ResourceConfigFileName = "resource_config.yaml"

// CommunicationConfigFileName is a name of botkube communication configuration file
var CommunicationConfigFileName = "comm_config.yaml"

// AccessConfigFileName is a name of botkube profile configuration file
var AccessConfigFileName = "access_config.yaml"

// Notify flag to toggle event notification
var Notify = true

// NotifType to change notification type
type NotifType string

// Config structure of configuration yaml file
type Config struct {
	Resources       []Resource
	Recommendations bool
	Communications  Communications
	Settings        Settings
}

// Resource contains resources to watch
type Resource struct {
	Name          string
	Namespaces    Namespaces
	Events        []EventType
	UpdateSetting UpdateSetting `yaml:"updateSetting"`
}

//UpdateSetting struct defines updateEvent fields specification
type UpdateSetting struct {
	Fields      []string
	IncludeDiff bool `yaml:"includeDiff"`
}

// Namespaces contains namespaces to include and ignore
// Include contains a list of namespaces to be watched,
//  - "all" to watch all the namespaces
// Ignore contains a list of namespaces to be ignored when all namespaces are included
// It is an optional (omitempty) field which is tandem with Include [all]
// example : include [all], ignore [x,y,z]
type Namespaces struct {
	Include []string
	Ignore  []string `yaml:",omitempty"`
}

// Communications channels to send events to
type Communications struct {
	Slack         Slack
	ElasticSearch ElasticSearch
	Mattermost    Mattermost
	Webhook       Webhook
}

// Slack configuration to authentication and send notifications
type Slack struct {
	Enabled        bool
	Channel        string
	NotifType      NotifType       `yaml:",omitempty"`
	Token          string          `yaml:",omitempty"`
	AccessBindings []AccessBinding `yaml:"accessBindings"`
}

// AccessBinding maps channel to profile
type AccessBinding struct {
	ChannelName  string `yaml:"channelName"`
	ProfileName  string `yaml:"profile"`
	ProfileValue Profile
}

// Profile defines access limititation for a specific channel
type Profile struct {
	Name                       string
	Namespaces                 []string `yaml:"namespaces"`
	Kubectl                    Kubectl
	AllowedKubectlResourceMap  map[string]bool
}

// AllProfiles contain all defined profiles
type AllProfiles struct {
	Profiles []Profile
}

// getProfile will return specific profile out of all defined profiles based on supplied profile name including correct value of AllowedKubectlResourceMap
func (all AllProfiles) getProfile(profileName string) (Profile, error) {
	p := Profile{}
	for _, profile := range all.Profiles {
		if profile.Name == profileName {
			allowedKubectlResourceMap := make(map[string]bool)
			for _, r := range profile.Kubectl.Commands.Resources {
				allowedKubectlResourceMap[r] = true
			}
			profile.AllowedKubectlResourceMap = allowedKubectlResourceMap
      p = profile
			return p, nil
		}
	}
	return p, errors.New("Selected profile not found in the provided profiles")
}

// ElasticSearch config auth settings
type ElasticSearch struct {
	Enabled    bool
	Username   string
	Password   string `yaml:",omitempty"`
	Server     string
	AWSSigning AWSSigning `yaml:"awsSigning"`
	Index      Index
}

// AWSSigning contains AWS configurations
type AWSSigning struct {
	Enabled   bool
	AWSRegion string `yaml:"awsRegion"`
	RoleArn   string `yaml:"roleArn"`
}

// Index settings for ELS
type Index struct {
	Name     string
	Type     string
	Shards   int
	Replicas int
}

// Mattermost configuration to authentication and send notifications
type Mattermost struct {
	Enabled        bool
	Channel        string
	URL            string
	Token          string
	Team           string
	NotifType      NotifType       `yaml:",omitempty"`
	AccessBindings []AccessBinding `yaml:"accessBindings"`
}

// Webhook configuration to send notifications
type Webhook struct {
	Enabled bool
	URL     string
}

// Kubectl configuration for executing commands inside cluster
type Kubectl struct {
	Enabled          bool
	Commands         Commands
	DefaultNamespace string `yaml:"defaultNamespace"`
	RestrictAccess   bool   `yaml:"restrictAccess"`
}

// Commands allowed in bot
type Commands struct {
	Verbs     []string
	Resources []string
}

// Settings for multicluster support
type Settings struct {
	ClusterName     string
	Kubectl         Kubectl
	ConfigWatcher   bool
	UpgradeNotifier bool `yaml:"upgradeNotifier"`
}

func (eventType EventType) String() string {
	return string(eventType)
}

// DefaultAccessBindings returns the default value of of single lenght AccessBindings by channel name
func DefaultAccessBindings(channel string, kubectlValue Kubectl) *[]AccessBinding {
	defaultValue := []AccessBinding{}
	profileValue := *new(Profile)
	profileValue.Name = "BOTKUBE_ADMIN"
	profileValue.Namespaces = []string{"default", "kube-system"}
	profileValue.Kubectl = kubectlValue
	accessBindingValue := new(AccessBinding)
	accessBindingValue.ChannelName = channel
	accessBindingValue.ProfileName = "BOTKUBE_ADMIN"
	accessBindingValue.ProfileValue = profileValue //{Name: "BOTKUBE_ADMIN", Namespaces: [kube-system],Kubectl: new(Kubectl)}
	defaultValue = append(defaultValue, *accessBindingValue)
	return &defaultValue
}

// New returns new Config
func New() (*Config, error) {
	c := &Config{}
	configPath := os.Getenv("CONFIG_PATH")
	resourceConfigFilePath := filepath.Join(configPath, ResourceConfigFileName)
	resourceConfigFile, err := os.Open(resourceConfigFilePath)
	defer resourceConfigFile.Close()
	if err != nil {
		return c, err
	}

	b, err := ioutil.ReadAll(resourceConfigFile)
	if err != nil {
		return c, err
	}

	if len(b) != 0 {
		err = yaml.Unmarshal(b, c)
		if err != nil {
			return c, err
		}
	}

	communicationConfigFilePath := filepath.Join(configPath, CommunicationConfigFileName)
	communicationConfigFile, err := os.Open(communicationConfigFilePath)
	defer communicationConfigFile.Close()
	if err != nil {
		return c, err
	}

	b, err = ioutil.ReadAll(communicationConfigFile)
	if err != nil {
		return c, err
	}

	if len(b) != 0 {
		err = yaml.Unmarshal(b, c)
		if err != nil {
			return c, err
		}
	}
	accessConfigFilePath := filepath.Join(configPath, AccessConfigFileName)
	accessConfigFile, err := os.Open(accessConfigFilePath)
	defer accessConfigFile.Close()
	if err != nil {
		log.Infof("AccessConfig file not found")

		if (c.Communications.Slack.Enabled && c.Communications.Slack.Channel != "") || (c.Communications.Mattermost.Enabled && c.Communications.Mattermost.Channel != "") {
			log.Warn("Configuration provided as used in older version of botkube")
			log.Warn("Check the updated configuration at: https://www.botkube.io/configuration/")
			log.Warn("setting default profile values to proceed")

			if c.Communications.Slack.Enabled {
				c.Communications.Slack.AccessBindings = *DefaultAccessBindings(c.Communications.Slack.Channel, c.Settings.Kubectl)
			}
			if c.Communications.Mattermost.Enabled {
				c.Communications.Mattermost.AccessBindings = *DefaultAccessBindings(c.Communications.Mattermost.Channel, c.Settings.Kubectl)
			}

			return c, nil
		} else if !c.Communications.Slack.Enabled && !c.Communications.Mattermost.Enabled {
			return c, nil
		} else {
			return c, err
		}
	}

	b, err = ioutil.ReadAll(accessConfigFile)
	if err != nil {
		return c, err
	}
	profiles := &AllProfiles{}
	if len(b) != 0 {
		err = yaml.Unmarshal(b, profiles)
		if err != nil {
			return c, err
		}
	}
	// Map right profile's value with config: For slack
	if c.Communications.Slack.Enabled {
		for i, accessBind := range c.Communications.Slack.AccessBindings {
			c.Communications.Slack.AccessBindings[i].ProfileValue, err = profiles.getProfile(accessBind.ProfileName)
			if err != nil {
				return c, err
			}
		}
	}

	// Map right profile's value with config: For mattermost
	if c.Communications.Mattermost.Enabled {
		for i, accessBind := range c.Communications.Mattermost.AccessBindings {
			c.Communications.Mattermost.AccessBindings[i].ProfileValue, err = profiles.getProfile(accessBind.ProfileName)
			if err != nil {
				return c, err
			}
		}
	}

	return c, nil
}

// GetAllProfiles returns All profiles specified in accessConfigFile
func GetAllProfiles() (*AllProfiles, error) {
	profiles := &AllProfiles{}
	configPath := os.Getenv("CONFIG_PATH")
	accessConfigFilePath := filepath.Join(configPath, AccessConfigFileName)
	accessConfigFile, err := os.Open(accessConfigFilePath)
	defer accessConfigFile.Close()
	if err != nil {
		log.Infof("AccessConfig file not found")
		return profiles, err
	}

	b, err := ioutil.ReadAll(accessConfigFile)
	if err != nil {
		return profiles, err
	}
	if len(b) != 0 {
		err = yaml.Unmarshal(b, profiles)
		if err != nil {
			return profiles, err
		}
		return profiles, nil
	}
	return profiles, err
}
