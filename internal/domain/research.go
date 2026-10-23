package domain

import "time"

type Activity struct { ID, SeedbedID, Title, Description, Status string; Date *time.Time; ResponsibleIDs, Evidence []string; CreatedAt, UpdatedAt time.Time }
type Project struct { ID, SeedbedID, Title, Summary, ResearchLine, Status string; ResponsibleIDs, MemberIDs []string; StartDate, EndDate *time.Time; Progress int; CreatedBy string; UpdatedAt time.Time }
type ProjectHistory struct { ID, ProjectID, AuthorID, Field, Value string; CreatedAt time.Time }
type ResearchProposal struct { ID, SeedbedID, Title, Problem, Objectives, Methodology, Status string; ResponsibleIDs, Attachments []string; AuthorID string; Observations string; CreatedAt, UpdatedAt time.Time; History []ProposalHistory }
type ProposalHistory struct { ID, ProposalID, AuthorID, Status, Observation string; CreatedAt time.Time }
type SeedbedReview struct { ID, SeedbedID, AuthorID, Text, Status, ModerationNote string; CreatedAt, ModeratedAt *time.Time }
