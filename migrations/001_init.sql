-- Minimal feedback service schema
-- Drop existing tables if they exist
DROP TABLE IF EXISTS feedback_files;

-- Create the single table for feedback metadata
CREATE TABLE feedback_files (
    id  INT NOT NULL PRIMARY KEY,
    user_id INT NOT NULL,
    lab_id INT,
    title VARCHAR(255) NOT NULL,
    created_at TIMESTAMP WITH TIME ZONE DEFAULT NOW(),
    updated_at TIMESTAMP WITH TIME ZONE DEFAULT NOW()
);

-- Create indexes for performance
CREATE INDEX idx_feedback_user_id ON feedback_files(user_id);
CREATE INDEX idx_feedback_lab_id ON feedback_files(lab_id);
CREATE INDEX idx_feedback_created_at ON feedback_files(created_at DESC);

-- Create trigger to automatically update updated_at
CREATE OR REPLACE FUNCTION update_updated_at_column()
RETURNS TRIGGER AS $$
BEGIN
    NEW.updated_at = NOW();
    RETURN NEW;
END;
$$ language 'plpgsql';

CREATE TRIGGER update_feedback_files_updated_at
    BEFORE UPDATE ON feedback_files
    FOR EACH ROW
    EXECUTE FUNCTION update_updated_at_column();
