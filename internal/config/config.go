package config

import (
	"os"
	"path/filepath"
	"time"

	"github.com/asianchinaboi/backendserver/internal/logger"
	"gopkg.in/yaml.v3"
)

type config struct {
	Guild  guild  `yaml:"guild"`
	User   user   `yaml:"user"`
	Server server `yaml:"server"`
}

type guild struct {
	MaxInvites   int           `yaml:"maxInvites"`
	MaxMsgLength int           `yaml:"maxMsgLength"`
	Timeout      time.Duration `yaml:"timeout"`
}

type user struct {
	MaxGuildsPerUser  int           `yaml:"maxGuildsPerUser"`  //not used yet
	MaxFriendsPerUser int           `yaml:"maxFriendsPerUser"` //not used yet
	CoolDownLength    time.Duration `yaml:"coolDownLength"`
	CoolDownTokens    int           `yaml:"coolDownTokens"`
	TokenExpireTime   time.Duration `yaml:"tokenExpireTime"`
	WSPerUser         int           `yaml:"wsPerUser"`
}

type server struct {
	Host               string        `yaml:"host"`
	Port               string        `yaml:"port"`
	Timeout            timeout       `yaml:"timeout"`
	BufferSize         bufferSize    `yaml:"bufferSize"`
	SnowflakeNodeID    int64         `yaml:"snowflakeNodeID"`
	TempFileAlive      time.Duration `yaml:"tempFileAlive"`
	ImageProfileSize   int           `yaml:"imageProfileSize"`
	MaxFileSize        int           `yaml:"maxFileSize"`
	MaxBodyRequestSize int           `yaml:"maxBodyRequestSize"`
}

type timeout struct {
	Server time.Duration `yaml:"server"`
	Write  time.Duration `yaml:"write"`
	Read   time.Duration `yaml:"read"`
	Idle   time.Duration `yaml:"idle"`
}

type bufferSize struct {
	Read  int `yaml:"read"`
	Write int `yaml:"write"`
}

var (
	Config *config
)

func loadConfig() (*config, error) {
	conf := &config{}
	path, err := os.Getwd()
	if err != nil {
		return nil, err
	}
	path = filepath.Join(path, "config.yml")
	file, err := os.OpenFile(path, os.O_RDONLY, 0644)
	if err != nil {
		return nil, err //make config if doesnt exist later - done
	}
	defer file.Close()
	d := yaml.NewDecoder(file)
	if err := d.Decode(&conf); err != nil {
		return nil, err
	}
	return conf, nil
}

func createConfig() (*config, error) {
	conf := &config{
		Guild: guild{
			MaxInvites:   10,
			MaxMsgLength: 2048,
			Timeout:      20 * time.Second,
		},
		User: user{
			MaxGuildsPerUser:  100,
			MaxFriendsPerUser: 200,
			CoolDownLength:    10 * time.Second,
			CoolDownTokens:    25,
			TokenExpireTime:   60 * time.Hour * 24,
			WSPerUser:         5,
		},
		Server: server{
			Host: "0.0.0.0",
			Port: "8080",
			Timeout: timeout{
				Server: 15 * time.Second,
				Write:  15 * time.Second,
				Read:   15 * time.Second,
				Idle:   15 * time.Second,
			},
			BufferSize: bufferSize{
				Read:  4096,
				Write: 4096,
			},
			SnowflakeNodeID:    1,
			TempFileAlive:      24 * time.Hour,
			ImageProfileSize:   4096,
			MaxFileSize:        1024 * 1024 * 15, // 15mb
			MaxBodyRequestSize: 1024 * 1024 * 5,  // 5mb
		},
	}
	path, err := os.Getwd()
	if err != nil {
		return nil, err
	}
	path = filepath.Join(path, "config.yml")
	file, err := os.Create(path)
	if err != nil {
		return nil, err
	}
	defer file.Close()
	e := yaml.NewEncoder(file)
	if err := e.Encode(conf); err != nil {
		return nil, err
	}
	return conf, nil
}

func init() {
	var err error
	Config, err = loadConfig()
	if err != nil { //have a look at this later
		Config, err = createConfig()
		if err != nil {
			logger.Fatal.Fatalln(err)
		}
		logger.Info.Println("Created config.yml")
	}
}
