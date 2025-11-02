// dto/user_subject_summary.go
package dto

// import (
// 	"time"

// 	"github.com/google/uuid"
// 	"gorm.io/datatypes"
// 	// sesuaikan bila path model berbeda
// )

// //
// // A. Komponen breakdown (JSONB)
// // ------------------------------------------------

// type BreakdownItem struct {
// 	Weight float64  `json:"weight" validate:"gte=0,lte=100"`                    // bobot persen
// 	Score  *float64 `json:"score,omitempty" validate:"omitempty,gte=0,lte=100"` // nilai 0..100 (nullable)
// }

// type Breakdown map[string]BreakdownItem // contoh: {"tugas":{...}, "kuis":{...}}

// //
// // B. Create / Update / Response DTOs
// // ------------------------------------------------

// // Create (backend-driven) — final_score, passed, breakdown boleh dikirim atau dihitung di service.
// type CreateUserSubjectSummaryDTO struct {
// 	SchoolID        uuid.UUID `json:"school_id"         validate:"required"`
// 	StudentID       uuid.UUID `json:"student_id"        validate:"required"`
// 	ClassSubjectsID uuid.UUID `json:"class_subjects_id" validate:"required"`

// 	CSSTID            *uuid.UUID `json:"csst_id,omitempty"`
// 	TermID            *uuid.UUID `json:"term_id,omitempty"`
// 	FinalAssessmentID *uuid.UUID `json:"final_assessment_id,omitempty"`

// 	FinalScore    *float64 `json:"final_score,omitempty" validate:"omitempty,gte=0,lte=100"`
// 	PassThreshold *float64 `json:"pass_threshold,omitempty" validate:"omitempty,gte=0,lte=100"` // default 70 jika nil
// 	Passed        *bool    `json:"passed,omitempty"`                                            // jika nil & final_score ada → dihitung

// 	Breakdown Breakdown `json:"breakdown,omitempty"`

// 	TotalAssessments       *int       `json:"total_assessments,omitempty"`
// 	TotalCompletedAttempts *int       `json:"total_completed_attempts,omitempty"`
// 	LastAssessedAt         *time.Time `json:"last_assessed_at,omitempty"`

// 	CertificateGenerated *bool   `json:"certificate_generated,omitempty"`
// 	Note                 *string `json:"note,omitempty"`
// }

// // Update (PATCH semantics) — semua optional pakai pointer
// type UpdateUserSubjectSummaryDTO struct {
// 	CSSTID            *uuid.UUID `json:"csst_id,omitempty"`
// 	TermID            *uuid.UUID `json:"term_id,omitempty"`
// 	FinalAssessmentID *uuid.UUID `json:"final_assessment_id,omitempty"`

// 	FinalScore    *float64 `json:"final_score,omitempty" validate:"omitempty,gte=0,lte=100"`
// 	PassThreshold *float64 `json:"pass_threshold,omitempty" validate:"omitempty,gte=0,lte=100"`
// 	Passed        *bool    `json:"passed,omitempty"`

// 	Breakdown *Breakdown `json:"breakdown,omitempty"`

// 	TotalAssessments       *int       `json:"total_assessments,omitempty"`
// 	TotalCompletedAttempts *int       `json:"total_completed_attempts,omitempty"`
// 	LastAssessedAt         *time.Time `json:"last_assessed_at,omitempty"`

// 	CertificateGenerated *bool   `json:"certificate_generated,omitempty"`
// 	Note                 *string `json:"note,omitempty"`
// }

// // Response (untuk API output)
// type UserSubjectSummaryResponse struct {
// 	ID uuid.UUID `json:"id"`

// 	SchoolID        uuid.UUID  `json:"school_id"`
// 	StudentID       uuid.UUID  `json:"student_id"`
// 	ClassSubjectsID uuid.UUID  `json:"class_subjects_id"`
// 	CSSTID          *uuid.UUID `json:"csst_id,omitempty"`
// 	TermID          *uuid.UUID `json:"term_id,omitempty"`

// 	FinalAssessmentID *uuid.UUID `json:"final_assessment_id,omitempty"`

// 	FinalScore    *float64 `json:"final_score,omitempty"`
// 	PassThreshold float64  `json:"pass_threshold"`
// 	Passed        bool     `json:"passed"`

// 	Breakdown Breakdown `json:"breakdown,omitempty"`

// 	TotalAssessments       *int       `json:"total_assessments,omitempty"`
// 	TotalCompletedAttempts *int       `json:"total_completed_attempts,omitempty"`
// 	LastAssessedAt         *time.Time `json:"last_assessed_at,omitempty"`

// 	CertificateGenerated bool    `json:"certificate_generated"`
// 	Note                 *string `json:"note,omitempty"`

// 	CreatedAt time.Time  `json:"created_at"`
// 	UpdatedAt time.Time  `json:"updated_at"`
// 	DeletedAt *time.Time `json:"deleted_at,omitempty"`
// }

// //
// // C. Filter & Pagination DTO
// // ------------------------------------------------

// type ListUserSubjectSummaryFilter struct {
// 	SchoolID        uuid.UUID  `form:"school_id"         validate:"required"`
// 	StudentID       *uuid.UUID `form:"student_id"`
// 	ClassSubjectsID *uuid.UUID `form:"class_subjects_id"`
// 	TermID          *uuid.UUID `form:"term_id"`
// 	Passed          *bool      `form:"passed"`

// 	MinScore *float64 `form:"min_score" validate:"omitempty,gte=0,lte=100"`
// 	MaxScore *float64 `form:"max_score" validate:"omitempty,gte=0,lte=100"`

// 	Page     int    `form:"page" validate:"gte=1"`
// 	PageSize int    `form:"page_size" validate:"gte=1,lte=200"`
// 	OrderBy  string `form:"order_by"` // contoh: "final_score desc", "updated_at desc"
// }

// //
// // D. Mapper Helpers (DTO ⇄ Model)
// // ------------------------------------------------

// // ToModel: Create DTO -> Model
// func (d CreateUserSubjectSummaryDTO) ToModel() models.UserSubjectSummary {
// 	// default threshold 70 jika tidak dikirim
// 	threshold := 70.0
// 	if d.PassThreshold != nil {
// 		threshold = *d.PassThreshold
// 	}

// 	// hitung passed bila nil dan final_score tersedia
// 	passed := false
// 	if d.Passed != nil {
// 		passed = *d.Passed
// 	} else if d.FinalScore != nil {
// 		passed = (*d.FinalScore) >= threshold
// 	}

// 	// certificate_generated default false bila nil
// 	cert := false
// 	if d.CertificateGenerated != nil {
// 		cert = *d.CertificateGenerated
// 	}

// 	var breakdown datatypes.JSONMap = nil
// 	if d.Breakdown != nil {
// 		// datatypes.JSONMap adalah alias map[string]any
// 		tmp := make(datatypes.JSONMap, len(d.Breakdown))
// 		for k, v := range d.Breakdown {
// 			item := map[string]any{
// 				"weight": v.Weight,
// 			}
// 			if v.Score != nil {
// 				item["score"] = *v.Score
// 			}
// 			tmp[k] = item
// 		}
// 		breakdown = tmp
// 	}

// 	return models.UserSubjectSummary{
// 		// PK by DB default
// 		UserSubjectSummarySchoolID:          d.SchoolID,
// 		UserSubjectSummarySchoolStudentID:   d.StudentID,
// 		UserSubjectSummaryClassSubjectsID:   d.ClassSubjectsID,
// 		UserSubjectSummaryCSSTID:            d.CSSTID,
// 		UserSubjectSummaryTermID:            d.TermID,
// 		UserSubjectSummaryFinalAssessmentID: d.FinalAssessmentID,

// 		UserSubjectSummaryFinalScore:    d.FinalScore,
// 		UserSubjectSummaryPassThreshold: threshold,
// 		UserSubjectSummaryPassed:        passed,

// 		UserSubjectSummaryBreakdown: breakdown,

// 		UserSubjectSummaryTotalAssessments:       d.TotalAssessments,
// 		UserSubjectSummaryTotalCompletedAttempts: d.TotalCompletedAttempts,
// 		UserSubjectSummaryLastAssessedAt:         d.LastAssessedAt,

// 		UserSubjectSummaryCertificateGenerated: cert,
// 		UserSubjectSummaryNote:                 d.Note,
// 	}
// }

// // PatchModel: Update DTO -> apply ke Model (PATCH semantics)
// func (d UpdateUserSubjectSummaryDTO) PatchModel(m *models.UserSubjectSummary) {
// 	if d.CSSTID != nil {
// 		m.UserSubjectSummaryCSSTID = d.CSSTID
// 	}
// 	if d.TermID != nil {
// 		m.UserSubjectSummaryTermID = d.TermID
// 	}
// 	if d.FinalAssessmentID != nil {
// 		m.UserSubjectSummaryFinalAssessmentID = d.FinalAssessmentID
// 	}

// 	if d.FinalScore != nil {
// 		m.UserSubjectSummaryFinalScore = d.FinalScore
// 	}
// 	if d.PassThreshold != nil {
// 		m.UserSubjectSummaryPassThreshold = *d.PassThreshold
// 	}
// 	if d.Passed != nil {
// 		m.UserSubjectSummaryPassed = *d.Passed
// 	}

// 	if d.Breakdown != nil {
// 		tmp := make(datatypes.JSONMap, len(*d.Breakdown))
// 		for k, v := range *d.Breakdown {
// 			item := map[string]any{
// 				"weight": v.Weight,
// 			}
// 			if v.Score != nil {
// 				item["score"] = *v.Score
// 			}
// 			tmp[k] = item
// 		}
// 		m.UserSubjectSummaryBreakdown = tmp
// 	}

// 	if d.TotalAssessments != nil {
// 		m.UserSubjectSummaryTotalAssessments = d.TotalAssessments
// 	}
// 	if d.TotalCompletedAttempts != nil {
// 		m.UserSubjectSummaryTotalCompletedAttempts = d.TotalCompletedAttempts
// 	}
// 	if d.LastAssessedAt != nil {
// 		m.UserSubjectSummaryLastAssessedAt = d.LastAssessedAt
// 	}
// 	if d.CertificateGenerated != nil {
// 		m.UserSubjectSummaryCertificateGenerated = *d.CertificateGenerated
// 	}
// 	if d.Note != nil {
// 		m.UserSubjectSummaryNote = d.Note
// 	}
// }

// // FromModel: Model -> Response DTO
// func FromModelUserSubjectSummary(m models.UserSubjectSummary) UserSubjectSummaryResponse {
// 	// convert JSONMap -> Breakdown (aman jika nil)
// 	var breakdown Breakdown = nil
// 	if m.UserSubjectSummaryBreakdown != nil {
// 		breakdown = Breakdown{}
// 		for k, anyVal := range m.UserSubjectSummaryBreakdown {
// 			// expect map[string]any{"weight":number,"score":number?}
// 			b := BreakdownItem{}
// 			if mp, ok := anyVal.(map[string]any); ok {
// 				if w, ok := mp["weight"]; ok {
// 					switch vv := w.(type) {
// 					case float64:
// 						b.Weight = vv
// 					case float32:
// 						b.Weight = float64(vv)
// 					case int:
// 						b.Weight = float64(vv)
// 					}
// 				}
// 				if s, ok := mp["score"]; ok {
// 					switch vv := s.(type) {
// 					case float64:
// 						val := vv
// 						b.Score = &val
// 					case float32:
// 						val := float64(vv)
// 						b.Score = &val
// 					case int:
// 						val := float64(vv)
// 						b.Score = &val
// 					}
// 				}
// 			}
// 			breakdown[k] = b
// 		}
// 	}

// 	return UserSubjectSummaryResponse{
// 		ID:              m.UserSubjectSummaryID,
// 		SchoolID:        m.UserSubjectSummarySchoolID,
// 		StudentID:       m.UserSubjectSummarySchoolStudentID,
// 		ClassSubjectsID: m.UserSubjectSummaryClassSubjectsID,
// 		CSSTID:          m.UserSubjectSummaryCSSTID,
// 		TermID:          m.UserSubjectSummaryTermID,

// 		FinalAssessmentID: m.UserSubjectSummaryFinalAssessmentID,

// 		FinalScore:    m.UserSubjectSummaryFinalScore,
// 		PassThreshold: m.UserSubjectSummaryPassThreshold,
// 		Passed:        m.UserSubjectSummaryPassed,

// 		Breakdown: breakdown,

// 		TotalAssessments:       m.UserSubjectSummaryTotalAssessments,
// 		TotalCompletedAttempts: m.UserSubjectSummaryTotalCompletedAttempts,
// 		LastAssessedAt:         m.UserSubjectSummaryLastAssessedAt,

// 		CertificateGenerated: m.UserSubjectSummaryCertificateGenerated,
// 		Note:                 m.UserSubjectSummaryNote,

// 		CreatedAt: m.UserSubjectSummaryCreatedAt,
// 		UpdatedAt: m.UserSubjectSummaryUpdatedAt,
// 		DeletedAt: m.UserSubjectSummaryDeletedAt,
// 	}
// }
