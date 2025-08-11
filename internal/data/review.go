package data

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"review/internal/biz"
	"review/internal/client/ai"
	"review/internal/data/model"
	"review/internal/data/query"
	"review/pkg/snowflake"
	"strconv"
	"strings"
	"time"

	"github.com/elastic/go-elasticsearch/v8/typedapi/types"
	"github.com/go-kratos/kratos/v2/log"
	"github.com/redis/go-redis/v9"
	"golang.org/x/sync/singleflight"
)

type reviewRepo struct {
	data *Data
	log  *log.Helper
	ai   *ai.AIClient
}

// NewReviewRepo 新建评论仓库
func NewReviewRepo(data *Data, logger log.Logger, ai *ai.AIClient) biz.ReviewRepo {
	return &reviewRepo{
		data: data,
		log:  log.NewHelper(logger),
		ai:   ai,
	}
}

// SaveReview 保存评论
func (r *reviewRepo) SaveReview(ctx context.Context, review *model.ReviewInfo) (*model.ReviewInfo, error) {
	// 1. 数据校验
	// 同一条订单如果已存在评论，则在原内容基础上追加新评论；否则创建新评论
	existingReviews, err := r.data.q.ReviewInfo.WithContext(ctx).Where(r.data.q.ReviewInfo.OrderID.Eq(review.OrderID)).Find()
	if err != nil {
		return nil, err
	}

	if len(existingReviews) > 0 {
		// 追加评论内容到原有评论
		existingReview := existingReviews[0]

		// 追加评论内容，添加时间戳和标识
		appendedContent := existingReview.Content + "\n\n" +
			"[追加评论 " + time.Now().Format("2006-01-02 15:04:05") + "]:\n" +
			review.Content

		// 更新评论内容和其他可能的字段
		_, err = r.data.q.ReviewInfo.WithContext(ctx).Where(r.data.q.ReviewInfo.ReviewID.Eq(existingReview.ReviewID)).Updates(map[string]interface{}{
			"content":       appendedContent,
			"score":         review.Score, // 更新评分（如果需要）
			"service_score": review.ServiceScore,
			"express_score": review.ExpressScore,
			"pic_info":      review.PicInfo,   // 更新图片信息（如果需要）
			"video_info":    review.VideoInfo, // 更新视频信息（如果需要）
		})
		if err != nil {
			return nil, errors.New("追加评论失败")
		}

		// 重新从数据库获取更新后的评论信息，以确保数据（特别是时间戳）是最新的
		updatedReview, err := r.data.q.ReviewInfo.WithContext(ctx).Where(r.data.q.ReviewInfo.ReviewID.Eq(existingReview.ReviewID)).First()
		if err != nil {
			// 即使这里失败，主流程也算成功，但需要记录错误
			r.log.WithContext(ctx).Errorf("failed to fetch updated review after appending, reviewID: %d, err: %v", existingReview.ReviewID, err)
			return existingReview, nil // 返回追加前的数据
		}
		// 异步处理
		go r.syncAndAudit(updatedReview)
		return updatedReview, nil
	} else {
		// 创建新评论
		err = r.data.q.ReviewInfo.WithContext(ctx).Create(review)
		if err != nil {
			return nil, errors.New("创建评论失败")
		}

		// 异步处理
		go r.syncAndAudit(review) // 此时的review对象包含了数据库生成的ID和时间戳
		return review, nil
	}
}

// syncAndAudit 封装了需要异步执行的同步和审核任务
func (r *reviewRepo) syncAndAudit(review *model.ReviewInfo) {
	// 为后台任务创建一个新的上下文
	ctx := context.Background()

	// 1. 先进行AI审核，审核过程会更新DB中的状态
	auditedReview, auditErr := r.AuditReview(ctx, &biz.AuditReviewParam{ReviewID: review.ReviewID})
	if auditErr != nil {
		r.log.WithContext(ctx).Errorf("Async AI audit failed for review ID %d: %v", review.ReviewID, auditErr)
		// 如果审核失败，我们仍然将最初的“待审核”状态的评论同步到ES，以确保其可被搜索到
		auditedReview = review
	} else {
		r.log.WithContext(ctx).Infof("Async AI audit successful for review ID: %d", review.ReviewID)
	}

	// 2. 将最终状态的评论同步到ES
	if err := r.SaveToES(ctx, auditedReview); err != nil {
		r.log.WithContext(ctx).Errorf("Async SaveToES failed for review ID %d: %v", auditedReview.ReviewID, err)
	} else {
		r.log.WithContext(ctx).Infof("Async SaveToES successful for review ID: %d", auditedReview.ReviewID)
	}
}

// SaveToES 保存到ES
func (r *reviewRepo) SaveToES(ctx context.Context, review *model.ReviewInfo) error {
	_, err := r.data.es.Index("review").
		Id(strconv.FormatInt(review.ReviewID, 10)).
		Request(review).
		Do(ctx)
	if err != nil {
		r.log.WithContext(ctx).Errorf("failed to save review to ES: %v", err)
	}
	return err
}

// 自动ai审核, 异步执行
func (r *reviewRepo) AutoAuditReview(reviewToAudit *model.ReviewInfo) {
	// 为后台任务创建一个新的上下文，因为原始上下文将在HTTP请求完成后被取消。
	auditCtx := context.Background()
	r.log.WithContext(auditCtx).Infof("Starting async AI audit for review ID: %d", reviewToAudit.ReviewID)

	_, auditErr := r.AuditReview(auditCtx, &biz.AuditReviewParam{
		ReviewID: reviewToAudit.ReviewID,
	})

	if auditErr != nil {
		r.log.WithContext(auditCtx).Errorf("Async AI audit failed for review ID %d: %v", reviewToAudit.ReviewID, auditErr)
	} else {
		r.log.WithContext(auditCtx).Infof("Async AI audit successful for review ID: %d", reviewToAudit.ReviewID)
	}
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
		return nil, fmt.Errorf("评论 (ID: %d) 不存在，无法回复", reply.ReviewID)
	}

	// 1.1 检查是否已经回复
	if review.HasReply == 1 {
		return nil, errors.New("该评论已回复，不能重复回复")
	}

	// 1.2 检查商家ID是否匹配
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

	// 2. 调用AI审核
	approved, reason, err := r.ai.ModerateText(ctx, review.Content)
	if err != nil {
		r.log.Errorf("AI审核失败: %v", err)
		return review, err
	}
	var status int32
	var remarks string
	if !approved {
		status = 30
		remarks = "AI审核不通过"
	} else {
		status = 20
		remarks = "AI审核通过"
	}
	_, err = r.data.q.ReviewInfo.WithContext(ctx).Where(r.data.q.ReviewInfo.ReviewID.Eq(param.ReviewID)).Updates(map[string]interface{}{
		"status":     status,
		"op_reason":  reason,
		"op_remarks": remarks,
		"update_by":  "Gemini",
		"update_at":  time.Now(),
	})
	if err != nil {
		return nil, err
	}

	// // 2. 更新评论审核信息
	// _, err = r.data.q.ReviewInfo.WithContext(ctx).Where(r.data.q.ReviewInfo.ReviewID.Eq(param.ReviewID)).Updates(map[string]interface{}{
	// 	"status":     param.Status,
	// 	"op_user":    param.OpUser,
	// 	"op_reason":  param.OpReason,
	// 	"op_remarks": param.OpRemarks,
	// 	"update_by":  param.OpUser,
	// })
	// if err != nil {
	// 	return nil, err
	// }

	// 3. 查询并返回更新后的评论信息
	return r.GetReviewByReviewID(ctx, param.ReviewID)
}

// AppealReview 申诉评论
func (r *reviewRepo) AppealReview(ctx context.Context, param *biz.AppealReviewParam) (*model.ReviewAppealInfo, error) {
	// 1. 数据校验
	// 1.1 评论存在性校验
	review, err := r.GetReviewByReviewID(ctx, param.ReviewID)
	if err != nil {
		return nil, fmt.Errorf("评论 (ID: %d) 不存在，无法申诉", param.ReviewID)
	}
	// 1.2 权限校验：商家只能申诉自己店铺的评论
	if review.StoreID != param.StoreID {
		return nil, errors.New("商家不能申诉其他商家的评论")
	}

	// 1.3 申诉状态校验：只有待审核(10)状态的申诉可以更新申诉，其他状态不允许重复申诉
	existingAppeals, err := r.data.q.ReviewAppealInfo.WithContext(ctx).Where(r.data.q.ReviewAppealInfo.ReviewID.Eq(param.ReviewID)).Find()
	if err != nil {
		return nil, errors.New("查询申诉记录失败")
	}
	if len(existingAppeals) > 0 {
		if existingAppeals[0].Status != 10 {
			return nil, errors.New("该评论存在已审核的申诉记录，不能重复申诉")
		}
	}

	// 2. 创建申诉记录
	// 2.1 如果已存在待审核状态的申诉记录，则使用该申诉ID，供更新这条申诉；否则生成新的申诉ID，供生成新的申诉
	var appealID int64
	if len(existingAppeals) > 0 {
		appealID = existingAppeals[0].AppealID
	} else {
		appealID = snowflake.GenID()
	}
	appeal := &model.ReviewAppealInfo{
		AppealID:  appealID,
		ReviewID:  param.ReviewID,
		StoreID:   param.StoreID,
		Status:    10, // 待审核状态
		Reason:    param.Reason,
		Content:   param.Content,
		PicInfo:   param.PicInfo,
		VideoInfo: param.VideoInfo,
	}

	// 3. 保存申诉记录
	// 如果已存在待审核状态的申诉记录，则更新；否则创建新记录
	if len(existingAppeals) > 0 {
		// 更新现有待审核状态的申诉记录
		_, err = r.data.q.ReviewAppealInfo.WithContext(ctx).Where(r.data.q.ReviewAppealInfo.AppealID.Eq(existingAppeals[0].AppealID)).Updates(map[string]interface{}{
			"reason":     appeal.Reason,
			"content":    appeal.Content,
			"pic_info":   appeal.PicInfo,
			"video_info": appeal.VideoInfo,
		})
		if err != nil {
			return nil, errors.New("更新申诉记录失败")
		}
		// 返回更新后的申诉记录
		appeal.AppealID = existingAppeals[0].AppealID
	} else {
		// 创建新的申诉记录
		err = r.data.q.ReviewAppealInfo.WithContext(ctx).Create(appeal)
		if err != nil {
			return nil, errors.New("创建申诉记录失败")
		}
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
			"reason":     param.OpReason,
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

// ListReviewByStoreID 根据商家ID获取评论列表（分页）
func (r *reviewRepo) ListReviewByStoreID(ctx context.Context, storeID int64, offset int32, limit int32) ([]*biz.MyReviewInfo, error) {
	return r.ListReviewByStoreID1(ctx, storeID, offset, limit)
}

func (r *reviewRepo) ListReviewByUserID(ctx context.Context, userID int64, offset int32, limit int32) ([]*biz.MyReviewInfo, error) {
	return r.ListReviewByUserID1(ctx, userID, offset, limit)
}

func (r *reviewRepo) ListReviewsByStatus(ctx context.Context, status int32, offset int32, limit int32) ([]*biz.MyReviewInfo, error) {
	// For simplicity, we create a new function for ES query by status, bypassing the generic cache layer for now.
	// A more robust implementation might involve a more flexible caching key.
	return r.listReviewsByStatusFromES(ctx, status, offset, limit)
}

// ListAppealsByStatus lists appeal records by status with pagination.
func (r *reviewRepo) ListAppealsByStatus(ctx context.Context, status int32, offset int32, limit int32) ([]*model.ReviewAppealInfo, error) {
	// Directly query DB for now. If needed, we can add ES indexing later for appeals.
	appeals, err := r.data.q.ReviewAppealInfo.WithContext(ctx).
		Where(r.data.q.ReviewAppealInfo.Status.Eq(status)).
		Offset(int(offset)).
		Limit(int(limit)).
		Find()
	if err != nil {
		return nil, err
	}
	return appeals, nil
}

var g = singleflight.Group{}

// 升级版带缓存的查询函数, 根据商家ID获取评论列表（分页）
func (r *reviewRepo) ListReviewByStoreID1(ctx context.Context, storeID int64, offset int32, limit int32) ([]*biz.MyReviewInfo, error) {
	// 1. 从redis中获取数据
	// 2. 如果redis中没有数据，则从ES中获取数据
	// 3. 通过singleflight.Group合并并发请求
	key := fmt.Sprintf("review:%d:%d:%d", storeID, offset, limit)
	b, err := r.GetDataBySingleFlight(ctx, key, "store")
	if err != nil {
		return nil, err
	}
	hm := new(types.HitsMetadata)
	if err := json.Unmarshal(b, hm); err != nil {
		return nil, err
	}
	// 4. 反序列化
	list := make([]*biz.MyReviewInfo, 0, hm.Total.Value)
	for _, hit := range hm.Hits {
		tmp := &biz.MyReviewInfo{}
		if err := json.Unmarshal(hit.Source_, tmp); err != nil {
			r.log.Errorf("es search result unmarshal error: %v", err)
			continue
		}
		list = append(list, tmp)
	}
	return list, nil
}

// 升级版带缓存的查询函数, 根据用户ID获取评论列表（分页）
func (r *reviewRepo) ListReviewByUserID1(ctx context.Context, userID int64, offset int32, limit int32) ([]*biz.MyReviewInfo, error) {
	// 1. 从redis中获取数据
	// 2. 如果redis中没有数据，则从ES中获取数据
	// 3. 通过singleflight.Group合并并发请求
	key := fmt.Sprintf("review:%d:%d:%d", userID, offset, limit)
	b, err := r.GetDataBySingleFlight(ctx, key, "user")
	if err != nil {
		return nil, err
	}
	hm := new(types.HitsMetadata)
	if err := json.Unmarshal(b, hm); err != nil {
		return nil, err
	}
	// 4. 反序列化
	list := make([]*biz.MyReviewInfo, 0, hm.Total.Value)
	for _, hit := range hm.Hits {
		tmp := &biz.MyReviewInfo{}
		if err := json.Unmarshal(hit.Source_, tmp); err != nil {
			r.log.Errorf("es search result unmarshal error: %v", err)
			continue
		}
		list = append(list, tmp)
	}
	return list, nil
}

// 通过singleflight获取数据
func (r *reviewRepo) GetDataBySingleFlight(ctx context.Context, key string, target string) ([]byte, error) {
	v, err, _ := g.Do(key, func() (interface{}, error) {
		// 1. 从redis中获取数据
		data, err := r.GetDataFromCache(ctx, key)
		if err == nil {
			r.log.Debugf("GetDataBySingleFlight(from redis cache), key: %s, data: %s", key, string(data))
			return data, nil
		}
		// 2. 如果redis中没有数据，则从ES中获取数据
		if errors.Is(err, redis.Nil) {
			data, err = r.GetDataFromES(ctx, key, target)
			if err == nil {
				r.log.Debugf("GetDataBySingleFlight(from es), key: %s, data: %s", key, string(data))
				return data, r.SetCache(ctx, key, data)
			}
			return nil, err
		}
		// 3. 如果查询redis报错，则返回错误
		return nil, err
	})
	if err != nil {
		return nil, err
	}
	return v.([]byte), nil
}

// 读缓存
func (r *reviewRepo) GetDataFromCache(ctx context.Context, key string) ([]byte, error) {
	r.log.Debugf("GetDataFromCache, key: %s", key)
	return r.data.rdb.Get(ctx, key).Bytes()
}

// 从ES中获取数据，target: store or user
func (r *reviewRepo) GetDataFromES(ctx context.Context, key string, target string) ([]byte, error) {
	values := strings.Split(key, ":")
	if len(values) < 4 {
		return nil, errors.New("key format error")
	}
	index := values[0]
	id := values[1] // storeID or userID
	offsetStr := values[2]
	limitStr := values[3]

	offset, err := strconv.Atoi(offsetStr)
	if err != nil {
		return nil, err
	}
	limit, err := strconv.Atoi(limitStr)
	if err != nil {
		return nil, err
	}

	// 去ES查询
	var fieldName string
	if target == "store" {
		fieldName = "store_id"
	} else if target == "user" {
		fieldName = "user_id"
	} else if target == "status" {
		fieldName = "status"
	} else {
		return nil, errors.New("invalid target")
	}

	resp, err := r.data.es.Search().
		Index(index).
		Query(&types.Query{
			Bool: &types.BoolQuery{
				Filter: []types.Query{
					{
						Term: map[string]types.TermQuery{
							fieldName: {Value: id},
						},
					},
				},
			},
		}).
		From(offset).
		Size(limit).
		Do(ctx)

	if err != nil {
		return nil, err
	}

	b, _ := json.Marshal(resp.Hits)

	return b, nil
}

// 设置缓存
func (r *reviewRepo) SetCache(ctx context.Context, key string, value []byte) error {
	return r.data.rdb.Set(ctx, key, value, time.Second*60).Err()
}

// listReviewsByStatusFromES directly queries Elasticsearch for reviews by their status.
func (r *reviewRepo) listReviewsByStatusFromES(ctx context.Context, status int32, offset int32, limit int32) ([]*biz.MyReviewInfo, error) {

	key := fmt.Sprintf("review:%d:%d:%d", status, offset, limit)
	b, err := r.GetDataBySingleFlight(ctx, key, "status")
	if err != nil {
		return nil, err
	}
	hm := new(types.HitsMetadata)
	if err := json.Unmarshal(b, hm); err != nil {
		return nil, err
	}
	// 4. 反序列化
	list := make([]*biz.MyReviewInfo, 0, hm.Total.Value)
	for _, hit := range hm.Hits {
		tmp := &biz.MyReviewInfo{}
		if err := json.Unmarshal(hit.Source_, tmp); err != nil {
			r.log.Errorf("es search result unmarshal error: %v", err)
			continue
		}
		list = append(list, tmp)
	}
	return list, nil
}

// // 旧版不带缓存的查询函数
// func (r *reviewRepo) ListReviewByStoreID2(ctx context.Context, storeID int64, offset int32, limit int32) ([]*biz.MyReviewInfo, error) {
// 	// 去ES查询
// 	resp, err := r.data.es.Search().Index("review").
// 		Query(&types.Query{
// 			Bool: &types.BoolQuery{
// 				Filter: []types.Query{
// 					{
// 						Term: map[string]types.TermQuery{
// 							"store_id": {Value: storeID},
// 						},
// 					},
// 				},
// 			},
// 		}).
// 		From(int(offset)).
// 		Size(int(limit)).
// 		Do(ctx)
// 	if err != nil {
// 		return nil, err
// 	}

// 	b, _ := json.Marshal(resp)
// 	fmt.Println("es search result total:", resp.Hits.Total.Value)
// 	fmt.Println("es search result hits:", string(b))

// 	list := make([]*biz.MyReviewInfo, 0, resp.Hits.Total.Value)
// 	// 反序列化
// 	for _, hit := range resp.Hits.Hits {
// 		tmp := &biz.MyReviewInfo{}
// 		if err := json.Unmarshal(hit.Source_, tmp); err != nil {
// 			r.log.Errorf("es search result unmarshal error: %v", err)
// 			continue
// 		}
// 		list = append(list, tmp)
// 	}
// 	return list, nil
// }

// // 旧版不带缓存的查询函数
// func (r *reviewRepo) ListReviewByUserID2(ctx context.Context, userID int64, offset int32, limit int32) ([]*biz.MyReviewInfo, error) {
// 	// 去ES查询
// 	resp, err := r.data.es.Search().Index("review").
// 		Query(&types.Query{
// 			Bool: &types.BoolQuery{
// 				Filter: []types.Query{
// 					{
// 						Term: map[string]types.TermQuery{
// 							"user_id": {Value: userID},
// 						},
// 					},
// 				},
// 			},
// 		}).
// 		From(int(offset)).
// 		Size(int(limit)).
// 		Do(ctx)
// 	if err != nil {
// 		return nil, err
// 	}

// 	b, _ := json.Marshal(resp)
// 	fmt.Println("es search result total:", resp.Hits.Total.Value)
// 	fmt.Println("es search result hits:", string(b))

// 	list := make([]*biz.MyReviewInfo, 0, resp.Hits.Total.Value)
// 	// 反序列化
// 	for _, hit := range resp.Hits.Hits {
// 		tmp := &biz.MyReviewInfo{}
// 		if err := json.Unmarshal(hit.Source_, tmp); err != nil {
// 			r.log.Errorf("es search result unmarshal error: %v", err)
// 			continue
// 		}
// 		list = append(list, tmp)
// 	}
// 	return list, nil
// }
