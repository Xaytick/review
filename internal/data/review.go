package data

import (
	"context"
	"errors"
	"review/internal/biz"
	"review/internal/data/model"
	"review/internal/data/query"
	"review/pkg/snowflake"

	"github.com/go-kratos/kratos/v2/log"
)

type reviewRepo struct {
	data *Data
	log  *log.Helper
}

// NewReviewRepo 新建评论仓库
func NewReviewRepo(data *Data, logger log.Logger) biz.ReviewRepo {
	return &reviewRepo{
		data: data,
		log:  log.NewHelper(logger),
	}
}

// SaveReview 保存评论
func (r *reviewRepo) SaveReview(ctx context.Context, review *model.ReviewInfo) (*model.ReviewInfo, error) {
	err := r.data.q.ReviewInfo.WithContext(ctx).Create(review)
	return review, err
}

// GetReviewByOrderID 根据订单ID查询评论
func (r *reviewRepo) GetReviewByOrderID(ctx context.Context, orderID int64) ([]*model.ReviewInfo, error) {
	return r.data.q.ReviewInfo.WithContext(ctx).Where(r.data.q.ReviewInfo.OrderID.Eq(orderID)).Find()
}

// SaveReply 保存回复
func (r *reviewRepo) SaveReply(ctx context.Context, reply *model.ReviewReplyInfo) (*model.ReviewReplyInfo, error) {
	// 1. 数据校验
	// 1.1 数据合法性校验：已回复的评论不能重复回复
	// 1.2 水平越权校验：商家不能回复其他商家的评论
	review, err := r.data.q.ReviewInfo.WithContext(ctx).Where(r.data.q.ReviewInfo.ReviewID.Eq(reply.ReviewID)).First()
	if err != nil {
		return nil, err
	}
	if review.HasReply == 1 {
		return nil, errors.New("已回复的评论不能重复回复")
	}
	if review.StoreID != reply.StoreID {
		return nil, errors.New("商家不能回复其他商家的评论")
	}
	// 2. 更新数据库中的数据，评价表和评价回复表要同时更新，涉及到事务操作
	r.data.q.Transaction(func(tx *query.Query) error {
		// 更新评价表has_reply字段
		if _, err := tx.ReviewInfo.WithContext(ctx).Where(tx.ReviewInfo.ReviewID.Eq(reply.ReviewID)).Update(
			tx.ReviewInfo.HasReply, 1); err != nil {
			return err
		}
		// 更新评价回复表
		if err := tx.ReviewReplyInfo.WithContext(ctx).Save(reply); err != nil {
			return err
		}
		return nil
	})
	// 3. 返回结果
	return reply, nil
}

// GetReviewByReviewID 根据评论ID查询评论
func (r *reviewRepo) GetReviewByReviewID(ctx context.Context, reviewID int64) (*model.ReviewInfo, error) {
	return r.data.q.ReviewInfo.WithContext(ctx).Where(r.data.q.ReviewInfo.ReviewID.Eq(reviewID)).First()
}

// AuditReview 审核评论
func (r *reviewRepo) AuditReview(ctx context.Context, param *biz.AuditReviewParam) (*model.ReviewInfo, error) {
	// 1. 数据校验
	// 评论状态校验：只有待审核状态(10)的评论才能进行审核
	review, err := r.data.q.ReviewInfo.WithContext(ctx).Where(r.data.q.ReviewInfo.ReviewID.Eq(param.ReviewID)).First()
	if err != nil {
		return nil, err
	}
	if review.Status != 10 {
		return nil, errors.New("只有待审核状态的评论才能进行审核")
	}

	// 2. 更新评论审核信息
	_, err = r.data.q.ReviewInfo.WithContext(ctx).Where(r.data.q.ReviewInfo.ReviewID.Eq(param.ReviewID)).Updates(map[string]interface{}{
		"status":     param.Status,
		"op_user":    param.OpUser,
		"op_reason":  param.OpReason,
		"op_remarks": param.OpRemarks,
		"update_by":  param.OpUser,
	})
	if err != nil {
		return nil, err
	}

	// 3. 查询并返回更新后的评论信息
	return r.GetReviewByReviewID(ctx, param.ReviewID)
}

// AppealReview 申诉评论
func (r *reviewRepo) AppealReview(ctx context.Context, param *biz.AppealReviewParam) (*model.ReviewAppealInfo, error) {
	// 1. 数据校验
	// 1.1 评论存在性校验
	review, err := r.GetReviewByReviewID(ctx, param.ReviewID)
	if err != nil {
		return nil, errors.New("无法获取评论信息")
	}
	// 1.2 权限校验：商家只能申诉自己店铺的评论
	if review.StoreID != param.StoreID {
		return nil, errors.New("商家不能申诉其他商家的评论")
	}
	// 1.3 申诉状态校验：一个评论只能申诉一次
	existingAppeals, err := r.data.q.ReviewAppealInfo.WithContext(ctx).Where(r.data.q.ReviewAppealInfo.ReviewID.Eq(param.ReviewID)).Find()
	if err != nil {
		return nil, errors.New("查询申诉记录失败")
	}
	if len(existingAppeals) > 0 {
		return nil, errors.New("该评论已存在申诉记录，不能重复申诉")
	}

	// 2. 创建申诉记录
	appeal := &model.ReviewAppealInfo{
		AppealID:  snowflake.GenID(),
		ReviewID:  param.ReviewID,
		StoreID:   param.StoreID,
		Status:    10, // 待审核状态
		Reason:    param.Reason,
		Content:   param.Content,
		PicInfo:   param.PicInfo,
		VideoInfo: param.VideoInfo,
		CreateBy:  "system",
		UpdateBy:  "system",
	}

	// 3. 保存申诉记录
	err = r.data.q.ReviewAppealInfo.WithContext(ctx).Create(appeal)
	if err != nil {
		return nil, errors.New("创建申诉记录失败")
	}

	// 4. 返回申诉信息
	return appeal, nil
}

// AuditAppeal 审核申诉
func (r *reviewRepo) AuditAppeal(ctx context.Context, param *biz.AuditAppealParam) (*model.ReviewAppealInfo, error) {
	// 1. 数据校验
	// 1.1 申诉记录存在性校验
	appeal, err := r.data.q.ReviewAppealInfo.WithContext(ctx).Where(r.data.q.ReviewAppealInfo.AppealID.Eq(param.AppealID)).First()
	if err != nil {
		return nil, errors.New("无法获取申诉记录")
	}
	// 1.2 申诉状态校验：只有待审核状态(10)的申诉才能进行审核
	if appeal.Status != 10 {
		return nil, errors.New("只有待审核状态的申诉才能进行审核")
	}

	// 2. 更新申诉记录和评论状态
	// 2.1 根据申诉审核结果确定申诉状态和评论状态
	var appeal_status, review_status int32
	switch param.Status {
	case 20: // 申诉通过
		appeal_status = 20 // 申诉通过状态
		review_status = 40 // 评论隐藏状态
	case 30: // 申诉驳回
		appeal_status = 30 // 申诉驳回状态
		review_status = 30 // 评论拒绝状态
	default:
		return nil, errors.New("无效的申诉审核状态")
	}
	// 2.2 原子操作更新申诉记录,同时更新评论状态
	err = r.data.q.Transaction(func(tx *query.Query) error {
		// 更新申诉记录
		_, err = tx.ReviewAppealInfo.WithContext(ctx).Where(tx.ReviewAppealInfo.AppealID.Eq(param.AppealID)).Updates(map[string]interface{}{
			"status":     appeal_status,
			"op_user":    param.OpUser,
			"reason":  	  param.OpReason,
			"op_remarks": param.OpRemarks,
			"update_by":  param.OpUser,
		})
		if err != nil {
			return err
		}

		// 更新评论状态
		_, err = tx.ReviewInfo.WithContext(ctx).Where(tx.ReviewInfo.ReviewID.Eq(appeal.ReviewID)).Updates(map[string]interface{}{
			"status":    review_status,
			"update_by": param.OpUser,
		})
		if err != nil {
			return err
		}
		return nil
	})
	if err != nil {
		return nil, errors.New("更新申诉记录和评论状态失败")
	}

	// 3. 查询并返回更新后的申诉信息
	updatedAppeal, err := r.data.q.ReviewAppealInfo.WithContext(ctx).Where(r.data.q.ReviewAppealInfo.AppealID.Eq(param.AppealID)).First()
	if err != nil {
		return nil, errors.New("查询更新后的申诉记录失败")
	}
	return updatedAppeal, nil
}

// ReplyReview 回复评论
func (r *reviewRepo) ReplyReview(ctx context.Context, param *biz.ReplyReviewParam) (*model.ReviewInfo, error) {
	// 1. 创建回复对象
	reply := &model.ReviewReplyInfo{
		ReplyID:   param.ReviewID,
		ReviewID:  param.ReviewID,
		StoreID:   param.StoreID,
		Content:   param.Content,
		PicInfo:   param.PicInfo,
		VideoInfo: param.VideoInfo,
	}

	// 2. 调用SaveReply保存回复
	_, err := r.SaveReply(ctx, reply)
	if err != nil {
		return nil, err
	}

	// 3. 返回更新后的评论信息
	return r.GetReviewByReviewID(ctx, param.ReviewID)
}
