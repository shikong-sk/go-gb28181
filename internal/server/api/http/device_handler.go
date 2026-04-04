package http

import (
	"net/http"
	"strconv"

	"git.skcks.cn/Shikong/go-gb28181/internal/server/model"
	"git.skcks.cn/Shikong/go-gb28181/internal/server/repository"
	"git.skcks.cn/Shikong/go-gb28181/internal/server/service"
	"git.skcks.cn/Shikong/go-gb28181/pkg/log"
	"github.com/gin-gonic/gin"
)

// DeviceHandler 设备 HTTP 处理器
type DeviceHandler struct {
	repo                       *repository.DeviceRepository
	catalogService             *service.CatalogService
	catalogSubscriptionService *service.CatalogSubscriptionService
}

// NewDeviceHandler 创建设备处理器
func NewDeviceHandler(repo *repository.DeviceRepository, catalogService *service.CatalogService, catalogSubscriptionService *service.CatalogSubscriptionService) *DeviceHandler {
	return &DeviceHandler{
		repo:                       repo,
		catalogService:             catalogService,
		catalogSubscriptionService: catalogSubscriptionService,
	}
}

// ListDevicesRequest 设备列表请求
type ListDevicesRequest struct {
	Page     int    `form:"page"`
	PageSize int    `form:"page_size"`
	Status   string `form:"status"`
}

// ListDevicesResponse 设备列表响应
type ListDevicesResponse struct {
	Total int64          `json:"total"`
	List  []model.Device `json:"list"`
}

// List 获取设备列表
func (h *DeviceHandler) List(c *gin.Context) {
	// 手动解析参数，兼容 camelCase 和 snake_case
	page, _ := strconv.Atoi(c.Query("page"))
	pageSize, _ := strconv.Atoi(c.Query("page_size"))
	if pageSize == 0 {
		pageSize, _ = strconv.Atoi(c.Query("pageSize")) // 兼容 camelCase
	}
	status := c.Query("status")

	// 默认值
	if page == 0 {
		page = 1
	}
	if pageSize == 0 {
		pageSize = 20
	}
	if pageSize > 100 {
		pageSize = 100
	}

	offset := (page - 1) * pageSize

	var devices []model.Device
	var total int64
	var err error

	if status != "" {
		devices, total, err = h.repo.ListByStatus(status, offset, pageSize)
	} else {
		devices, total, err = h.repo.List(offset, pageSize)
	}

	if err != nil {
		log.Error().Err(err).Msg("获取设备列表失败")
		c.JSON(http.StatusInternalServerError, Response{
			Code:    http.StatusInternalServerError,
			Message: "获取设备列表失败",
		})
		return
	}

	c.JSON(http.StatusOK, Response{
		Code:    http.StatusOK,
		Message: "success",
		Data: ListDevicesResponse{
			Total: total,
			List:  devices,
		},
	})
}

// GetDevice 获取设备详情
func (h *DeviceHandler) GetDevice(c *gin.Context) {
	deviceID := c.Param("device_id")
	if deviceID == "" {
		c.JSON(http.StatusBadRequest, Response{
			Code:    http.StatusBadRequest,
			Message: "设备ID不能为空",
		})
		return
	}

	device, err := h.repo.GetByDeviceID(deviceID)
	if err != nil {
		log.Error().Err(err).Str("device_id", deviceID).Msg("获取设备详情失败")
		c.JSON(http.StatusNotFound, Response{
			Code:    http.StatusNotFound,
			Message: "设备不存在",
		})
		return
	}

	c.JSON(http.StatusOK, Response{
		Code:    http.StatusOK,
		Message: "success",
		Data:    device,
	})
}

// DeleteDevice 删除设备
func (h *DeviceHandler) DeleteDevice(c *gin.Context) {
	deviceID := c.Param("device_id")
	if deviceID == "" {
		c.JSON(http.StatusBadRequest, Response{
			Code:    http.StatusBadRequest,
			Message: "设备ID不能为空",
		})
		return
	}

	if err := h.repo.Delete(deviceID); err != nil {
		log.Error().Err(err).Str("device_id", deviceID).Msg("删除设备失败")
		c.JSON(http.StatusInternalServerError, Response{
			Code:    http.StatusInternalServerError,
			Message: "删除设备失败",
		})
		return
	}

	c.JSON(http.StatusOK, Response{
		Code:    http.StatusOK,
		Message: "删除成功",
	})
}

// DeviceStatsResponse 设备统计响应
type DeviceStatsResponse struct {
	Total   int64 `json:"total"`
	Online  int64 `json:"online"`
	Offline int64 `json:"offline"`
}

// Stats 获取设备统计
func (h *DeviceHandler) Stats(c *gin.Context) {
	total, err := h.repo.CountTotal()
	if err != nil {
		log.Error().Err(err).Msg("统计设备总数失败")
		c.JSON(http.StatusInternalServerError, Response{
			Code:    http.StatusInternalServerError,
			Message: "统计设备失败",
		})
		return
	}

	online, err := h.repo.CountOnline()
	if err != nil {
		log.Error().Err(err).Msg("统计在线设备失败")
		c.JSON(http.StatusInternalServerError, Response{
			Code:    http.StatusInternalServerError,
			Message: "统计设备失败",
		})
		return
	}

	c.JSON(http.StatusOK, Response{
		Code:    http.StatusOK,
		Message: "success",
		Data: DeviceStatsResponse{
			Total:   total,
			Online:  online,
			Offline: total - online,
		},
	})
}

// SyncCatalog 同步设备目录
func (h *DeviceHandler) SyncCatalog(c *gin.Context) {
	deviceID := c.Param("device_id")
	if deviceID == "" {
		c.JSON(http.StatusBadRequest, Response{
			Code:    http.StatusBadRequest,
			Message: "设备ID不能为空",
		})
		return
	}

	// 检查设备是否存在
	_, err := h.repo.GetByDeviceID(deviceID)
	if err != nil {
		c.JSON(http.StatusNotFound, Response{
			Code:    http.StatusNotFound,
			Message: "设备不存在",
		})
		return
	}

	// 触发目录同步
	if h.catalogService == nil {
		c.JSON(http.StatusServiceUnavailable, Response{
			Code:    http.StatusServiceUnavailable,
			Message: "目录同步服务不可用",
		})
		return
	}

	if err := h.catalogService.SyncCatalog(deviceID); err != nil {
		log.Error().Err(err).Str("device_id", deviceID).Msg("同步设备目录失败")
		c.JSON(http.StatusInternalServerError, Response{
			Code:    http.StatusInternalServerError,
			Message: "同步设备目录失败: " + err.Error(),
		})
		return
	}

	c.JSON(http.StatusOK, Response{
		Code:    http.StatusOK,
		Message: "目录同步请求已发送",
	})
}

// Response 统一响应格式
type Response struct {
	Code    int         `json:"code"`
	Message string      `json:"message"`
	Data    interface{} `json:"data,omitempty"`
}

// ParsePagination 解析分页参数
func ParsePagination(page, pageSize string) (int, int) {
	p, _ := strconv.Atoi(page)
	if p < 1 {
		p = 1
	}
	ps, _ := strconv.Atoi(pageSize)
	if ps < 1 {
		ps = 20
	}
	if ps > 100 {
		ps = 100
	}
	return p, ps
}
