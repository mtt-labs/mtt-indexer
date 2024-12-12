package config

var Cfg Conf

type Conf struct {
	Port      int    `yaml:"port"`
	DbTailFix string `yaml:"db_tail_fix"`
	Rpc       string `yaml:"rpc"`
}
