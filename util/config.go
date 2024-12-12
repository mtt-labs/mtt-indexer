package util

import (
	"github.com/shopspring/decimal"
	"math/big"
	"os"

	"github.com/sirupsen/logrus"
	"gopkg.in/yaml.v2"
)

func LoadConfig(path string, config interface{}) {
	buf, err := os.ReadFile(path)
	if err != nil {
		logrus.WithFields(logrus.Fields{"err": err, "path": path}).Fatal("fail to read config")
	}

	if err = yaml.Unmarshal(buf, config); err != nil {
		logrus.WithField("err", err).Fatal("fail to parse config yaml")
	}
}

func ToNumeric(i *big.Int) decimal.Decimal {
	num := decimal.NewFromBigInt(i, 0)
	return num
}
