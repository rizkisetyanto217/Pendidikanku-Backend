package dto

type SurveyQuestionResponse struct {
	SurveyQuestionID         int      `json:"survey_question_id"`
	SurveyQuestionText       string   `json:"survey_question_text"`
	SurveyQuestionAnswer     []string `json:"survey_question_answer,omitempty"`
	SurveyQuestionOrderIndex int      `json:"survey_question_order_index"`
}
