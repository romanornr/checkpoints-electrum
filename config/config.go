package config

import (
	"gopkg.in/yaml.v2"
	"io/ioutil"
	"log"
)

type Conf struct {
	Host        string `yaml:"host"`
	RpcPort     string `yaml:"rpc_port"`
	RpcUsername string `yaml:"rpc_username"`
	RpcPassword string `yaml:"rpc_password"`
}

func (c *Conf) GetConf() *Conf {
	yamlFile, err := ioutil.ReadFile("config.yml")
	if err != nil {
		log.Printf("File config.yml doesn't exist: %s\n", err)
	}

	err = yaml.Unmarshal(yamlFile, c)
	if err != nil {
		log.Fatalf("Error unmarshal yaml file: %v\n", err)
	}
	return c
}
