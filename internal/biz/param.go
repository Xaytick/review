package biz

type AuditReviewParam struct {
	ReviewID int64
	Status int32
	OpUser string
	OpReason string
	OpRemarks string
}

type AppealReviewParam struct {	
	ReviewID int64
	StoreID int64
	Reason string
	Content string
	PicInfo string
	VideoInfo string
}

type AuditAppealParam struct {
	AppealID int64
	Status int32
	OpUser string
	OpReason string
	OpRemarks string
}

type ReplyReviewParam struct {
	ReviewID int64
	StoreID int64
	Content string
	PicInfo string
	VideoInfo string
}