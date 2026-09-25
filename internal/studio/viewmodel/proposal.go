package viewmodel

import "github.com/voocel/ainovel-cli/internal/studio/v2"

// V2 Proposal 类型直接复用治理层结构，避免 Bridge 再复制一套事实模型。
type Proposal = v2.Proposal
type ProposalChange = v2.ProposalChange
type EvidenceRef = v2.EvidenceRef
type VersionSnapshot = v2.VersionSnapshot
type CreateProposalRequest = v2.CreateProposalRequest
type ProposalChangeInput = v2.ProposalChangeInput
type EvidenceInput = v2.EvidenceInput
type CreateVersionRequest = v2.CreateVersionRequest
type ApplyResult = v2.ApplyResult
