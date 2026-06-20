package util

import (
	"encoding/json"
)

// NatsMessagePub 每个穿越微服务边界的NATS推送消息，都需要遵循如下的JSON格式
type NatsMessagePub struct {
	MessageULID    string          `json:"message_ulid"`    // 消息ULID
	SourceService  string          `json:"source_service"`  // 消息来源服务，如 gexam、gmall、gmail
	MessageType    string          `json:"message_type"`    // 消息类型，由来源服务定义
	MessagePayload json.RawMessage `json:"message_payload"` // 原始消息体 JSON
}

type NatsGExamStatusUpdateEvent struct {
	ExamID         string `json:"exam_id"`
	BusinessUnit   string `json:"business_unit"`
	CourseUnitUlid string `json:"course_unit_ulid"`
	ExamStatus     string `json:"exam_status"`
	ResultStatus   string `json:"result_status"`
	EventType      string `json:"event_type"`
	Timestamp      string `json:"timestamp"`
}

type NatsGMailStatusUpdateEvent struct {
	MailID       string `json:"mail_id"`
	BusinessUnit string `json:"business_unit"`
	Status       string `json:"status"`
	Timestamp    string `json:"timestamp"`
}

type NatsGPayStatusUpdateEvent struct {
	OrderID      string `json:"order_id"`
	BusinessUnit string `json:"business_unit"`
	Status       string `json:"status"`
	Timestamp    string `json:"timestamp"`
}

type NatsGCredsApplicationStatusUpdateEvent struct {
	ApplicationID     string `json:"application_id"`
	CandidateID       string `json:"candidate_id"`
	CatalogID         string `json:"catalog_id"`
	ApplicationStatus string `json:"application_status"`
	CredentialID      string `json:"credential_id"`
	Approved          bool   `json:"approved"`
	BusinessUnit      string `json:"business_unit"`
	Timestamp         string `json:"timestamp"`
}

type NatsGMallStageStatusEvent struct {
	PipelineULID string `json:"pipeline_ulid"`
	StageULID    string `json:"stage_ulid"`
	Timestamp    string `json:"timestamp"`
}

type NatsGCredsPdfStatusUpdateEvent struct {
	RequestID    string `json:"request_id"`
	BusinessUnit string `json:"business_unit"`
	Status       string `json:"status"`
	CredID       string `json:"cred_id,omitempty"`
	PdfFileHash  string `json:"pdf_file_hash,omitempty"`
	ErrorMessage string `json:"error_message,omitempty"`
	Timestamp    string `json:"timestamp"`
}

type NatsGMallOrderStatusUpdateEvent struct {
	EntityType       string `json:"entity_type"`
	OrderULID        string `json:"order_ulid"`
	CandidateID      string `json:"candidate_id"`
	PipelineCcULID   string `json:"pipeline_cc_ulid"`
	StageCcULID      string `json:"stage_cc_ulid"`
	CourseUnitULID   string `json:"course_unit_ulid"`
	CourseUnitCcULID string `json:"course_unit_cc_ulid"`
	Status           string `json:"status"`
	Reason           string `json:"reason"`
	Timestamp        string `json:"timestamp"`
}

// NatsGlmsCourseCompletedEvent is published by glms when a candidate completes a course.
type NatsGlmsCourseCompletedEvent struct {
	CandidateID string `json:"candidate_id"` // 考生 ULID
	CourseID    string `json:"course_id"`    // GLMS 课程版本 ULID (对应 gcc unit config 中的 glms_course_id)
	CourseTitle string `json:"course_title"` // 课程标题（仅用于日志）
	ProgressPct int32  `json:"progress_pct"` // 完成进度百分比
	CompletedAt string `json:"completed_at"` // 完成时间 RFC3339
}

// NatsGPaySubscriptionRenewedEvent is published when a subscription payment succeeds (renewal or recovery)
type NatsGPaySubscriptionRenewedEvent struct {
	StripeSubscriptionID string `json:"stripe_subscription_id"`
	StripeInvoiceID      string `json:"stripe_invoice_id"`
	StripePriceID        string `json:"stripe_price_id"`
	CurrentPeriodStart   int64  `json:"current_period_start"`
	CurrentPeriodEnd     int64  `json:"current_period_end"`
	AmountPaid           int64  `json:"amount_paid"`
	Currency             string `json:"currency"`
	Timestamp            string `json:"timestamp"`
}

// NatsGPaySubscriptionPaymentFailedEvent is published when a subscription payment attempt fails
type NatsGPaySubscriptionPaymentFailedEvent struct {
	StripeSubscriptionID string `json:"stripe_subscription_id"`
	StripeInvoiceID      string `json:"stripe_invoice_id"`
	StripePriceID        string `json:"stripe_price_id"`
	CurrentPeriodStart   int64  `json:"current_period_start"`
	CurrentPeriodEnd     int64  `json:"current_period_end"`
	AmountDue            int64  `json:"amount_due"`
	Currency             string `json:"currency"`
	Timestamp            string `json:"timestamp"`
}

// NatsGPaySubscriptionCancelledEvent is published when a subscription is deleted/cancelled
type NatsGPaySubscriptionCancelledEvent struct {
	StripeSubscriptionID string `json:"stripe_subscription_id"`
	CanceledAt           int64  `json:"canceled_at"`
	EndedAt              int64  `json:"ended_at"`
	Timestamp            string `json:"timestamp"`
}
