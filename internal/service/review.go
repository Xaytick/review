package service

import (
	"context"
	"fmt"

	pb "review/api/review/v1"
	"review/internal/biz"
	"review/internal/data/model"
	"review/pkg/snowflake"
)

type ReviewService struct {
	pb.UnimplementedReviewServer
	uc *biz.ReviewUsecase
}

func NewReviewService(uc *biz.ReviewUsecase) *ReviewService {
	return &ReviewService{uc: uc}
}

// CreateReview 创建评论
func (s *ReviewService) CreateReview(ctx context.Context, req *pb.CreateReviewRequest) (*pb.CreateReviewReply, error) {
	fmt.Println("[service] CreateReview, req:", req)
	// 调用biz层
	var anonymous int32
	if req.Anonymous {
		anonymous = 1
	} else {
		anonymous = 0
	}

	// 处理ID：优先使用req中的参数，如果没有则使用snowflake生成
	var reviewID, userID, orderID, storeID int64
	// req中没有reviewID，直接使用雪花算法生成
	reviewID = snowflake.GenID()

	if req.UserID > 0 {
		userID = req.UserID
	} else {
		userID = snowflake.GenID()
	}

	if req.OrderID > 0 {
		orderID = req.OrderID
	} else {
		orderID = snowflake.GenID()
	}

	if req.StoreID > 0 {
		storeID = req.StoreID
	} else {
		storeID = snowflake.GenID()
	}
	review, err := s.uc.CreateReview(ctx, &model.ReviewInfo{
		ReviewID:     reviewID,
		UserID:       userID,
		OrderID:      orderID,
		StoreID:      storeID,
		Score:        req.Score,
		ServiceScore: req.ServiceScore,
		ExpressScore: req.ExpressScore,
		Content:      req.Content,
		PicInfo:      req.PicInfo,
		VideoInfo:    req.VideoInfo,
		Status:       10, // Default status to "Pending"
		Anonymous:    anonymous,
	})
	if err != nil {
		return nil, err
	}
	// 拼装返回值
	return &pb.CreateReviewReply{ReviewInfo: &pb.ReviewInfo{
		ReviewID:     review.ReviewID,
		UserID:       review.UserID,
		OrderID:      review.OrderID,
		StoreID:      review.StoreID,
		Score:        review.Score,
		ServiceScore: review.ServiceScore,
		ExpressScore: review.ExpressScore,
		Content:      review.Content,
		PicInfo:      review.PicInfo,
		VideoInfo:    review.VideoInfo,
		Status:       review.Status,
	}}, nil
}

// GetReview 获取评论
func (s *ReviewService) GetReview(ctx context.Context, req *pb.GetReviewRequest) (*pb.GetReviewReply, error) {
	fmt.Println("[service] GetReview, req:", req)
	// 调用biz层
	review, err := s.uc.GetReview(ctx, req.ReviewID)
	if err != nil {
		return nil, err
	}
	// 拼装返回值
	return &pb.GetReviewReply{ReviewInfo: &pb.ReviewInfo{
		ReviewID:     review.ReviewID,
		UserID:       review.UserID,
		OrderID:      review.OrderID,
		Score:        review.Score,
		ServiceScore: review.ServiceScore,
		ExpressScore: review.ExpressScore,
		Content:      review.Content,
		PicInfo:      review.PicInfo,
		VideoInfo:    review.VideoInfo,
		Status:       review.Status,
	}}, nil
}

// AuditReview 审核评论
func (s *ReviewService) AuditReview(ctx context.Context, req *pb.AuditReviewRequest) (*pb.AuditReviewReply, error) {
	fmt.Println("[service] AuditReview, req:", req)
	// 调用biz层
	var opRemarks string
	if req.OpRemarks != nil {
		opRemarks = *req.OpRemarks
	}
	review, err := s.uc.AuditReview(ctx, &biz.AuditReviewParam{
		ReviewID:  req.ReviewID,
		Status:    req.Status,
		OpUser:    req.OpUser,
		OpReason:  req.OpReason,
		OpRemarks: opRemarks,
	})
	if err != nil {
		return nil, err
	}
	// 拼装返回值
	return &pb.AuditReviewReply{ReviewID: review.ReviewID, Status: review.Status}, nil
}

// ReplyReview 回复评论
func (s *ReviewService) ReplyReview(ctx context.Context, req *pb.ReplyReviewRequest) (*pb.ReplyReviewReply, error) {
	fmt.Println("[service] ReplyReview, req:", req)
	// 调用biz层
	reply, err := s.uc.ReplyReview(ctx, &biz.ReplyReviewParam{
		ReviewID:  req.ReviewID,
		StoreID:   req.StoreID,
		Content:   req.Content,
		PicInfo:   req.PicInfo,
		VideoInfo: req.VideoInfo,
	})
	if err != nil {
		return nil, err
	}
	// 拼装返回值
	return &pb.ReplyReviewReply{ReplyID: reply.ReplyID}, nil
}

// AppealReview 申诉评论
func (s *ReviewService) AppealReview(ctx context.Context, req *pb.AppealReviewRequest) (*pb.AppealReviewReply, error) {
	fmt.Println("[service] AppealReview, req:", req)
	// 调用biz层
	review, err := s.uc.AppealReview(ctx, &biz.AppealReviewParam{
		ReviewID:  req.ReviewID,
		StoreID:   req.StoreID,
		Reason:    req.Reason,
		Content:   req.Content,
		PicInfo:   req.PicInfo,
		VideoInfo: req.VideoInfo,
	})
	if err != nil {
		return nil, err
	}
	// 拼装返回值
	return &pb.AppealReviewReply{AppealID: snowflake.GenID(), Status: review.Status}, nil
}

// AuditAppeal 审核申诉
func (s *ReviewService) AuditAppeal(ctx context.Context, req *pb.AuditAppealRequest) (*pb.AuditAppealReply, error) {
	fmt.Println("[service] AuditAppeal, req:", req)
	// 调用biz层
	var opRemarks string
	if req.OpRemarks != nil {
		opRemarks = *req.OpRemarks
	}
	review, err := s.uc.AuditAppeal(ctx, &biz.AuditAppealParam{
		AppealID:  req.AppealID,
		Status:    req.Status,
		OpUser:    req.OpUser,
		OpReason:  req.OpReason,
		OpRemarks: opRemarks,
	})
	if err != nil {
		return nil, err
	}
	return &pb.AuditAppealReply{AppealID: req.AppealID, Status: review.Status}, nil
}

// ListReviewByStoreID 根据商家ID获取评论列表（分页）
func (s *ReviewService) ListReviewByStoreID(ctx context.Context, req *pb.ListReviewByStoreIDRequest) (*pb.ListReviewByStoreIDReply, error) {
	fmt.Println("[service] ListReviewByStoreID, req:", req)
	// 调用biz层
	reviews, err := s.uc.ListReviewByStoreID(ctx, req.StoreID, req.Page, req.Size)
	if err != nil {
		return nil, err
	}
	// 拼装返回值
	list := make([]*pb.ReviewInfo, 0, len(reviews))
	for _, review := range reviews {
		list = append(list, &pb.ReviewInfo{
			ReviewID:     review.ReviewID,
			UserID:       review.UserID,
			OrderID:      review.OrderID,
			StoreID:      review.StoreID,
			Score:        review.Score,
			ServiceScore: review.ServiceScore,
			ExpressScore: review.ExpressScore,
			Content:      review.Content,
			PicInfo:      review.PicInfo,
			VideoInfo:    review.VideoInfo,
			Status:       review.Status,
		})
	}
	return &pb.ListReviewByStoreIDReply{List: list}, nil
}

// ListReviewByUserID 根据用户ID获取评论列表（分页）
func (s *ReviewService) ListReviewByUserID(ctx context.Context, req *pb.ListReviewByUserIDRequest) (*pb.ListReviewByUserIDReply, error) {
	fmt.Println("[service] ListReviewByUserID, req:", req)
	// 调用biz层
	reviews, err := s.uc.ListReviewByUserID(ctx, req.UserID, req.Page, req.Size)
	if err != nil {
		return nil, err
	}
	// 拼装返回值
	list := make([]*pb.ReviewInfo, 0, len(reviews))
	for _, review := range reviews {
		list = append(list, &pb.ReviewInfo{
			ReviewID:     review.ReviewID,
			UserID:       review.UserID,
			OrderID:      review.OrderID,
			StoreID:      review.StoreID,
			Score:        review.Score,
			ServiceScore: review.ServiceScore,
			ExpressScore: review.ExpressScore,
			Content:      review.Content,
			PicInfo:      review.PicInfo,
			VideoInfo:    review.VideoInfo,
			Status:       review.Status,
		})
	}
	return &pb.ListReviewByUserIDReply{List: list}, nil
}

// ListReviewsByStatus retrieves a list of reviews by status with pagination.
func (s *ReviewService) ListReviewsByStatus(ctx context.Context, req *pb.ListReviewsByStatusRequest) (*pb.ListReviewByUserIDReply, error) {
	fmt.Println("[service] ListReviewsByStatus, req:", req)
	// Call the biz layer
	reviews, err := s.uc.ListReviewsByStatus(ctx, req.Status, req.Page, req.Size)
	if err != nil {
		return nil, err
	}
	// Assemble the response
	list := make([]*pb.ReviewInfo, 0, len(reviews))
	for _, review := range reviews {
		list = append(list, &pb.ReviewInfo{
			ReviewID:     review.ReviewID,
			UserID:       review.UserID,
			OrderID:      review.OrderID,
			StoreID:      review.StoreID,
			Score:        review.Score,
			ServiceScore: review.ServiceScore,
			ExpressScore: review.ExpressScore,
			Content:      review.Content,
			PicInfo:      review.PicInfo,
			VideoInfo:    review.VideoInfo,
			Status:       review.Status,
		})
	}
	// Note: We are reusing ListReviewByUserIDReply as the response message.
	return &pb.ListReviewByUserIDReply{List: list}, nil
}

// ListAppealsByStatus retrieves a list of appeals by status with pagination.
func (s *ReviewService) ListAppealsByStatus(ctx context.Context, req *pb.ListAppealsByStatusRequest) (*pb.ListAppealsByStatusReply, error) {
	fmt.Println("[service] ListAppealsByStatus, req:", req)
	appeals, err := s.uc.ListAppealsByStatus(ctx, req.Status, req.Page, req.Size)
	if err != nil {
		return nil, err
	}
	list := make([]*pb.AppealInfo, 0, len(appeals))
	for _, a := range appeals {
		list = append(list, &pb.AppealInfo{
			AppealID:  a.AppealID,
			ReviewID:  a.ReviewID,
			StoreID:   a.StoreID,
			Status:    a.Status,
			Reason:    a.Reason,
			Content:   a.Content,
			PicInfo:   a.PicInfo,
			VideoInfo: a.VideoInfo,
		})
	}
	return &pb.ListAppealsByStatusReply{List: list}, nil
}
