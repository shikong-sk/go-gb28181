package http

import (
	"net/http"
	"strconv"

	"git.skcks.cn/Shikong/go-gb28181/internal/server/service"
	"git.skcks.cn/Shikong/go-gb28181/pkg/log"
	"github.com/gin-gonic/gin"
)

// AlarmHandler 报警 HTTP 处理器
type AlarmHandler struct {
	alarmService *service.AlarmService
}

// NewAlarmHandler 创建报警处理器
func NewAlarmHandler(alarmService *service.AlarmService) *AlarmHandler {
	return &AlarmHandler{alarmService: alarmService}
}

// ListAlarmsRequest 报警列表请求
type ListAlarmsRequest struct {
	Page      int    `form:"page"`
	PageSize  int    `form:"page_size"`
	DeviceID  string `form:"device_id"`
	Priority  string `form:"priority"`
	StartTime string `form:"start_time"`
	EndTime   string `form:"end_time"`
}

// ListAlarmsResponse 报警列表响应
type ListAlarmsResponse struct {
	Total int64       `json:"total"`
	List  []AlarmItem `json:"list"`
}

// AlarmItem 报警项
type AlarmItem struct {
	ID               uint   `json:"id"`
	DeviceID         string `json:"deviceId"`
	AlarmPriority    string `json:"alarmPriority"`
	AlarmMethod      string `json:"alarmMethod"`
	AlarmTime        string `json:"alarmTime"`
	AlarmDescription string `json:"alarmDescription"`
	CreatedAt        string `json:"createdAt"`
}

// List 获取报警列表
func (h *AlarmHandler) List(c *gin.Context) {
	var req ListAlarmsRequest
	if err := c.ShouldBindQuery(&req); err != nil {
		c.JSON(http.StatusBadRequest, Response{
			Code:    http.StatusBadRequest,
			Message: "参数错误: " + err.Error(),
		})
		return
	}

	alarms, total, err := h.alarmService.ListAlarms(
		req.DeviceID, req.Priority, req.StartTime, req.EndTime,
		req.Page, req.PageSize,
	)
	if err != nil {
		log.Error().Err(err).Msg("获取报警列表失败")
		c.JSON(http.StatusInternalServerError, Response{
			Code:    http.StatusInternalServerError,
			Message: "获取报警列表失败",
		})
		return
	}

	items := make([]AlarmItem, len(alarms))
	for i, alarm := range alarms {
		items[i] = AlarmItem{
			ID:               alarm.ID,
			DeviceID:         alarm.DeviceID,
			AlarmPriority:    alarm.AlarmPriority,
			AlarmMethod:      alarm.AlarmMethod,
			AlarmTime:        alarm.AlarmTime,
			AlarmDescription: alarm.AlarmDescription,
			CreatedAt:        alarm.CreatedAt.Format("2006-01-02 15:04:05"),
		}
	}

	c.JSON(http.StatusOK, Response{
		Code:    http.StatusOK,
		Message: "success",
		Data: ListAlarmsResponse{
			Total: total,
			List:  items,
		},
	})
}

// GetAlarm 获取报警详情
func (h *AlarmHandler) GetAlarm(c *gin.Context) {
	idStr := c.Param("id")
	id, err := strconv.ParseUint(idStr, 10, 32)
	if err != nil {
		c.JSON(http.StatusBadRequest, Response{
			Code:    http.StatusBadRequest,
			Message: "无效的报警ID",
		})
		return
	}

	alarm, err := h.alarmService.GetAlarm(uint(id))
	if err != nil {
		c.JSON(http.StatusNotFound, Response{
			Code:    http.StatusNotFound,
			Message: "报警不存在",
		})
		return
	}

	c.JSON(http.StatusOK, Response{
		Code:    http.StatusOK,
		Message: "success",
		Data:    alarm,
	})
}

// DeleteAlarm 删除报警
func (h *AlarmHandler) DeleteAlarm(c *gin.Context) {
	idStr := c.Param("id")
	id, err := strconv.ParseUint(idStr, 10, 32)
	if err != nil {
		c.JSON(http.StatusBadRequest, Response{
			Code:    http.StatusBadRequest,
			Message: "无效的报警ID",
		})
		return
	}

	if err := h.alarmService.DeleteAlarm(uint(id)); err != nil {
		log.Error().Err(err).Uint("id", uint(id)).Msg("删除报警失败")
		c.JSON(http.StatusInternalServerError, Response{
			Code:    http.StatusInternalServerError,
			Message: "删除报警失败",
		})
		return
	}

	c.JSON(http.StatusOK, Response{
		Code:    http.StatusOK,
		Message: "删除成功",
	})
}

// DeleteAll 清空所有报警记录
// DeleteAll 清空所有报警记录
func (h *AlarmHandler) DeleteAll(c *gin.Context) {
	if err := h.alarmService.DeleteAll(); err != nil {
		log.Error().Err(err).Msg("清空报警记录失败")
		c.JSON(http.StatusInternalServerError, Response{
			Code:    http.StatusInternalServerError,
			Message: "清空报警记录失败",
		})
		return
	}

	c.JSON(http.StatusOK, Response{
		Code:    http.StatusOK,
		Message: "清空成功",
	})
}

// GetConfig 获取报警配置
func (h *AlarmHandler) GetConfig(c *gin.Context) {
	c.JSON(http.StatusOK, Response{
		Code:    http.StatusOK,
		Message: "success",
		Data: gin.H{
			"enabled":        h.alarmService.IsEnabled(),
			"retention_days": h.alarmService.GetRetentionDays(),
		},
	})
}
