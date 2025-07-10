CREATE TABLE IF NOT EXISTS survey_questions (
    survey_question_id SERIAL PRIMARY KEY,
    survey_question_text TEXT NOT NULL,
    survey_question_answer TEXT[] DEFAULT NULL, -- NULL jika open-ended
    survey_question_order_index INT NOT NULL,

    created_at TIMESTAMP WITHOUT TIME ZONE DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP WITHOUT TIME ZONE DEFAULT CURRENT_TIMESTAMP
);


CREATE TABLE IF NOT EXISTS user_surveys (
    user_survey_id SERIAL PRIMARY KEY,
    user_survey_user_id UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    user_survey_question_id INT NOT NULL REFERENCES survey_questions(survey_question_id) ON DELETE CASCADE,
    user_survey_answer TEXT NOT NULL,

    created_at TIMESTAMP WITHOUT TIME ZONE DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP WITHOUT TIME ZONE DEFAULT CURRENT_TIMESTAMP
);


CREATE TABLE IF NOT EXISTS test_exams (
    test_exam_id SERIAL PRIMARY KEY,
    test_exam_name VARCHAR(50) NOT NULL,
    test_exam_status VARCHAR(10) CHECK (test_exam_status IN ('active', 'pending', 'archived')) DEFAULT 'pending',

    created_at TIMESTAMP WITHOUT TIME ZONE DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP WITHOUT TIME ZONE DEFAULT CURRENT_TIMESTAMP
);


CREATE TABLE IF NOT EXISTS user_test_exams (
    user_test_exam_id SERIAL PRIMARY KEY,
    user_test_exam_user_id UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    user_test_exam_test_exam_id INT NOT NULL REFERENCES test_exams(test_exam_id) ON DELETE CASCADE,

    user_test_exam_percentage_grade INTEGER NOT NULL DEFAULT 0,
    user_test_exam_time_duration INTEGER NOT NULL DEFAULT 0,

    created_at TIMESTAMP WITHOUT TIME ZONE DEFAULT CURRENT_TIMESTAMP
);