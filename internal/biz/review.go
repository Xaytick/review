package biz

import (
	"context"
	"errors"
	"strings"
	"time"

	v1 "review/api/review/v1"
	"review/internal/data/model"
	"review/pkg/snowflake"

	"github.com/go-kratos/kratos/v2/log"
)

type ReviewRepo interface {
	SaveReview(context.Context, *model.ReviewInfo) (*model.ReviewInfo, error)
	SaveReply(context.Context, *model.ReviewReplyInfo) (*model.ReviewReplyInfo, error)
	GetReviewByOrderID(context.Context, int64) ([]*model.ReviewInfo, error)
	GetReviewByReviewID(context.Context, int64) (*model.ReviewInfo, error)
	AuditReview(context.Context, *AuditReviewParam) (*model.ReviewInfo, error)
	AppealReview(context.Context, *AppealReviewParam) (*model.ReviewAppealInfo, error)
	AuditAppeal(context.Context, *AuditAppealParam) (*model.ReviewAppealInfo, error)
	ReplyReview(context.Context, *ReplyReviewParam) (*model.ReviewInfo, error)
	ListReviewByStoreID(context.Context, int64, int32, int32) ([]*MyReviewInfo, error)
	ListReviewByUserID(context.Context, int64, int32, int32) ([]*MyReviewInfo, error)
}

type ReviewUsecase struct {
	repo ReviewRepo
	log  *log.Helper
}

func NewReviewUsecase(repo ReviewRepo, logger log.Logger) *ReviewUsecase {
	return &ReviewUsecase{
		repo: repo,
		log:  log.NewHelper(logger),
	}
}

// 自定义评论信息, 用于解决 unmarshal error: parsing time "2025-07-03 22:58:19" as "2006-01-02T15:04:05Z07:00"
// 使用string类型解决 error: json: cannot unmarshal string into Go struct field MyReviewInfo.ReviewInfo.anonymous of type int32
type MyReviewInfo struct {
	*model.ReviewInfo
	CreateAt MyTime `json:"create_at"`
	UpdateAt MyTime `json:"update_at"`
	ID             int64      `json:"id,string"`
	Version        int32      `json:"version,string"`       
	ReviewID       int64      `json:"review_id,string"`          
	Score          int32      `json:"score,string"`         
	ServiceScore   int32      `json:"service_score,string"` 
	ExpressScore   int32      `json:"express_score,string"` 
	HasMedia       int32      `json:"has_media,string"`     
	OrderID        int64      `json:"order_id,string"`      
	SkuID          int64      `json:"sku_id,string"`        
	SpuID          int64      `json:"spu_id,string"`        
	StoreID        int64      `json:"store_id,string"`      
	UserID         int64      `json:"user_id,string"`       
	Anonymous      int32      `json:"anonymous,string"`     
	Status         int32      `json:"status,string"`        
	IsDefault      int32      `json:"is_default,string"`    
	HasReply       int32      `json:"has_reply,string"`     
}

// 自定义时间类型，便于实现UnmarshalJSON方法
type MyTime time.Time

// UnmarshalJSON 自定义时间反序列化，解决es和go时间格式不一样的问题
func (t *MyTime) UnmarshalJSON(data []byte) error {
	// 2025-07-03 22:58:19
	// 2025-07-03T22:58:19+08:00

	s := strings.Trim(string(data), `"`)
	tmp, err := time.ParseInLocation(time.DateTime, s, time.Local)
	if err != nil {
		return err
	}
	*t = MyTime(tmp)
	return nil
	
}

// 创建评论, service层调用
func (uc *ReviewUsecase) CreateReview(ctx context.Context, review *model.ReviewInfo) (*model.ReviewInfo, error) {
	uc.log.WithContext(ctx).Debugf("[biz] CreateReview, review: %v", review)
	// 1. 数据校验
	_, err := uc.repo.GetReviewByOrderID(ctx, review.OrderID)
	if err != nil {
		return nil, v1.ErrorDbFailed("数据库查询评论失败, orderID: %d", review.OrderID)
	}
	// if len(reviews) > 0 {
	// 	return nil, v1.ErrorOrderReviewed("已评价的订单不能重复评价, orderID: %d", review.OrderID)
	// }

	// 2. 拼装数据入库
	return uc.repo.SaveReview(ctx, review)
}

// GetReview 获取评论
func (uc *ReviewUsecase) GetReview(ctx context.Context, reviewID int64) (*model.ReviewInfo, error) {
	uc.log.WithContext(ctx).Debugf("[biz] GetReview, reviewID: %d", reviewID)
	return uc.repo.GetReviewByReviewID(ctx, reviewID)
}

// AuditReview 审核评论
func (uc *ReviewUsecase) AuditReview(ctx context.Context, param *AuditReviewParam) (*model.ReviewInfo, error) {
	uc.log.WithContext(ctx).Debugf("[biz] AuditReview, param: %v", param)
	// 1. 数据校验
	review, err := uc.repo.GetReviewByReviewID(ctx, param.ReviewID)
	if err != nil {
		return nil, errors.New("无法获取评论信息")
	}
	if review.Status != 10 {
		return nil, errors.New("评论状态无法审核")
	}

	return uc.repo.AuditReview(ctx, param)
}

// AppealReview 申诉评论
func (uc *ReviewUsecase) AppealReview(ctx context.Context, param *AppealReviewParam) (*model.ReviewAppealInfo, error) {
	uc.log.WithContext(ctx).Debugf("[biz] AppealReview, param: %v", param)

	// 1. 业务参数校验

	// 2. 检查评论是否存在且状态可申诉
	review, err := uc.repo.GetReviewByReviewID(ctx, param.ReviewID)
	if err != nil {
		return nil, errors.New("无法获取评论信息")
	}

	// 评论必须是已发布状态才能申诉（假设20为已发布状态）
	if review.Status != 20 {
		return nil, errors.New("只有已发布的评论才能申诉")
	}

	// 3. 调用 data 层进行申诉
	return uc.repo.AppealReview(ctx, param)
}

// AuditAppeal 审核申诉
func (uc *ReviewUsecase) AuditAppeal(ctx context.Context, param *AuditAppealParam) (*model.ReviewAppealInfo, error) {
	uc.log.WithContext(ctx).Debugf("[biz] AuditAppeal, param: %v", param)

	// 1. 业务参数校验
	if param.Status != 20 && param.Status != 30 {
		return nil, errors.New("审核状态无效，只能设置为通过(20)或驳回(30)")
	}

	// 2. 调用 data 层进行审核
	return uc.repo.AuditAppeal(ctx, param)
}

// ReplyReview 回复评论
func (uc *ReviewUsecase) ReplyReview(ctx context.Context, param *ReplyReviewParam) (*model.ReviewReplyInfo, error) {
	uc.log.WithContext(ctx).Debugf("[biz] ReplyReview, param: %v", param)
	reply := &model.ReviewReplyInfo{
		ReplyID:   snowflake.GenID(),
		ReviewID:  param.ReviewID,
		StoreID:   param.StoreID,
		Content:   param.Content,
		PicInfo:   param.PicInfo,
		VideoInfo: param.VideoInfo,
	}
	return uc.repo.SaveReply(ctx, reply)
}

// ListReviewByStoreID 根据商家ID获取评论列表（分页）
func (uc *ReviewUsecase) ListReviewByStoreID(ctx context.Context, storeID int64, page int32, size int32) ([]*MyReviewInfo, error) {
	if page <= 0 {
		page = 1
	}
	if size <= 0 || size > 50 {
		size = 10
	}
	offset := (page - 1) * size
	limit := size

	uc.log.WithContext(ctx).Debugf("[biz] ListReviewByStoreID, storeID: %d, offset: %d, limit: %d", storeID, offset, limit)
	return uc.repo.ListReviewByStoreID(ctx, storeID, offset, limit)
}


// ListReviewByUserID 根据用户ID获取评论列表（分页）
func (uc *ReviewUsecase) ListReviewByUserID(ctx context.Context, userID int64, page int32, size int32) ([]*MyReviewInfo, error) {
	if page <= 0 {
		page = 1
	}
	if size <= 0 || size > 50 {
		size = 10
	}
	offset := (page - 1) * size
	limit := size
	uc.log.WithContext(ctx).Debugf("[biz] ListReviewByUserID, userID: %d, offset: %d, limit: %d", userID, offset, limit)
	return uc.repo.ListReviewByUserID(ctx, userID, offset, limit)
}
