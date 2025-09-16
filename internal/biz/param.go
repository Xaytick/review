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

type ListReviewsParam struct {
	UserID    int64
	StoreID   int64
	Status    int32
	MinScore  int32
	MaxScore  int32
	Keyword   string
	Page      int32
	Size      int32
	SortBy    string // create_at|score|like_count
	SortOrder string // asc|desc
}