package controller

import (
	"fmt"
	"github.com/gin-gonic/gin"
	"mtt-indexer/logger"
	"mtt-indexer/service"
	"net/http"
	"strconv"
)

const (
	ResponseCodeOk          = 200
	ResponseCodeParamsError = 50001
)

type Response struct {
	Code  int         `json:"code"`
	Msg   string      `json:"msg"`
	Data  interface{} `json:"data"`
	Total int         `json:"total"`
}

type Endpoint func(c *gin.Context)

type DelegatorListResp struct {
	Delegator  string   `json:"delegator"`
	Validators []string `json:"validators"`
	Amounts    []string `json:"amounts"`
	Denom      string   `json:"denom"`
}

func DelegatorListEndpoint(s service.IService) gin.HandlerFunc {
	return func(c *gin.Context) {
		delegator, exist := c.GetQuery("delegator")
		if !exist {
			resp := &Response{
				Code: ResponseCodeParamsError,
				Msg:  "",
				Data: "",
			}
			c.JSON(http.StatusOK, resp)
			return
		}

		record, err := s.GetDelegatorList(delegator)
		if err != nil {
			logger.Logger.Errorf("Get delegator list error: %v", err)
			return
		}
		result := DelegatorListResp{
			Delegator:  record.Delegator,
			Validators: record.Validators,
			Amounts:    record.Amounts,
			Denom:      record.Denom,
		}
		resp := &Response{
			Code: ResponseCodeOk,
			Msg:  "",
			Data: result,
		}
		c.JSON(http.StatusOK, resp)
		return
	}
}

type History struct {
	Delegator      string `json:"delegator"`
	Validator      string `json:"validator"`
	Amount         string `json:"amount"`
	Denom          string `json:"denom"`
	TxHash         string `json:"tx_hash"`
	DelegationType uint8  `json:"delegation_type"` // true: add, false: rm
	DelegationTime int64  `json:"delegation_time"`
}

func DelegatorHistoryEndpoint(s service.IService) gin.HandlerFunc {
	return func(c *gin.Context) {
		delegator, exist := c.GetQuery("delegator")
		if !exist {
			resp := &Response{
				Code: ResponseCodeParamsError,
				Msg:  "",
				Data: "",
			}
			c.JSON(http.StatusOK, resp)
			return
		}
		limitStr, _ := c.GetQuery("limit")
		limit, _ := strconv.Atoi(limitStr)
		offsetStr, _ := c.GetQuery("offset")
		offset, _ := strconv.Atoi(offsetStr)

		ascStr, _ := c.GetQuery("asc")
		asc := false
		if ascStr == "true" {
			asc = true
		}

		records, total, err := s.GetDelegatorHistory(delegator, validLimit(limit, 20, 100), validOffset(offset), asc)
		if err != nil {
			logger.Logger.Error(fmt.Sprintf("GetRecord endpoint error : %s", err))
			return
		}

		result := []*History{}

		for _, record := range records {
			result = append(result, &History{
				Delegator:      record.Delegator,
				Validator:      record.Validator,
				Amount:         record.Amount,
				Denom:          record.Denom,
				TxHash:         record.TxHash,
				DelegationType: uint8(record.DelegationType),
				DelegationTime: record.DelegationTime.Unix(),
			})
		}

		resp := &Response{
			Code:  ResponseCodeOk,
			Msg:   "",
			Data:  result,
			Total: total,
		}
		c.JSON(http.StatusOK, resp)
		return
	}
}

func ValidatorHistoryEndpoint(s service.IService) gin.HandlerFunc {
	return func(c *gin.Context) {
		validator, exist := c.GetQuery("validator")
		if !exist {
			resp := &Response{
				Code: ResponseCodeParamsError,
				Msg:  "",
				Data: "",
			}
			c.JSON(http.StatusOK, resp)
			return
		}
		limitStr, _ := c.GetQuery("limit")
		limit, _ := strconv.Atoi(limitStr)
		offsetStr, _ := c.GetQuery("offset")
		offset, _ := strconv.Atoi(offsetStr)

		ascStr, _ := c.GetQuery("asc")
		asc := false
		if ascStr == "true" {
			asc = true
		}

		records, total, err := s.GetValidatorHistory(validator, validLimit(limit, 20, 100), validOffset(offset), asc)
		if err != nil {
			logger.Logger.Errorf("GetRecord endpoint error : %s", err)
			return
		}

		result := []*History{}

		for _, record := range records {
			result = append(result, &History{
				Delegator:      record.Delegator,
				Validator:      record.Validator,
				Amount:         record.Amount,
				Denom:          record.Denom,
				TxHash:         record.TxHash,
				DelegationType: uint8(record.DelegationType),
				DelegationTime: record.DelegationTime.Unix(),
			})
		}

		resp := &Response{
			Code:  ResponseCodeOk,
			Msg:   "",
			Data:  result,
			Total: total,
		}
		c.JSON(http.StatusOK, resp)
		return
	}
}

type RewardHistory struct {
	Amount string `json:"amount"`
	Time   int64  `json:"time"`
}

func RewardHistoryEndpoint(s service.IService) gin.HandlerFunc {
	return func(c *gin.Context) {
		validatorStr, exist := c.GetQuery("validator")
		if !exist {
			return
		}
		limitStr, _ := c.GetQuery("limit")
		limit, _ := strconv.Atoi(limitStr)
		offsetStr, _ := c.GetQuery("offset")
		offset, _ := strconv.Atoi(offsetStr)

		records, total, err := s.GetRewardHistory(validatorStr, validLimit(limit, 20, 100), validOffset(offset))
		if err != nil {
			logger.Logger.Errorf("GetRecord endpoint error : %s", err)
			return
		}

		result := []*RewardHistory{}

		for _, record := range records {
			result = append(result, &RewardHistory{
				Amount: record.Amount,
				Time:   record.Time,
			})
		}

		resp := &Response{
			Code:  ResponseCodeOk,
			Msg:   "",
			Data:  result,
			Total: total,
		}
		c.JSON(http.StatusOK, resp)
		return
	}
}

type Commission struct {
	Validator  string  `json:"validator"`
	Commission float64 `json:"commission"`
	Time       int64   `json:"time"`
}

func CommissionRecordEndpoint(s service.IService) gin.HandlerFunc {
	return func(c *gin.Context) {
		validator, exist := c.GetQuery("validator")
		if !exist {
			resp := &Response{
				Code: ResponseCodeParamsError,
				Msg:  "",
				Data: "",
			}
			c.JSON(http.StatusOK, resp)
			return
		}
		limitStr, _ := c.GetQuery("limit")
		limit, _ := strconv.Atoi(limitStr)
		offsetStr, _ := c.GetQuery("offset")
		offset, _ := strconv.Atoi(offsetStr)

		records, total, err := s.GetCommissionRecord(validator, validLimit(limit, 20, 100), validOffset(offset))
		if err != nil {
			logger.Logger.Errorf("GetRecord endpoint error : %s", err)
			return
		}

		result := []*Commission{}

		for _, record := range records {
			result = append(result, &Commission{
				Validator:  record.Validator,
				Commission: record.Commission,
				Time:       record.Time.Unix(),
			})
		}

		resp := &Response{
			Code:  ResponseCodeOk,
			Msg:   "",
			Data:  result,
			Total: total,
		}
		c.JSON(http.StatusOK, resp)
		return
	}
}

func HeightEndpoint(s service.IService) gin.HandlerFunc {
	return func(c *gin.Context) {
		height, err := s.GetChainHeight()
		if err != nil {
			logger.Logger.Errorf("GetChainHeight error : %s", err)
			return
		}

		resp := &Response{
			Code: ResponseCodeOk,
			Msg:  "",
			Data: height,
		}
		c.JSON(http.StatusOK, resp)
		return
	}
}

func validLimit(originLimit, defaultLimit, maxLimit int) int {
	if originLimit == 0 {
		return defaultLimit
	}
	if originLimit > maxLimit {
		return maxLimit
	}
	return originLimit
}

func validOffset(offset int) int {
	if offset < 0 {
		return 0
	}
	return offset
}
