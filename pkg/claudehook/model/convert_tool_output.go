package model

import (
	sdktypes "github.com/ngicks/crabswarm/pkg/api/gen/proto/go/sdk_types/v1"
)

func ToolOutputFromProto(pb *sdktypes.ToolOutput) *ToolOutput {
	if pb == nil {
		return nil
	}
	m := &ToolOutput{}
	switch v := pb.GetOutput().(type) {
	case *sdktypes.ToolOutput_Task:
		m.Task = TaskOutputFromProto(v.Task)
	case *sdktypes.ToolOutput_AskUserQuestion:
		m.AskUserQuestion = AskUserQuestionOutputFromProto(v.AskUserQuestion)
	case *sdktypes.ToolOutput_Bash:
		m.Bash = BashOutputFromProto(v.Bash)
	case *sdktypes.ToolOutput_BashOutput:
		m.BashOutput = BashOutputToolOutputFromProto(v.BashOutput)
	case *sdktypes.ToolOutput_Edit:
		m.Edit = EditOutputFromProto(v.Edit)
	case *sdktypes.ToolOutput_Read:
		m.Read = ReadOutputFromProto(v.Read)
	case *sdktypes.ToolOutput_Write:
		m.Write = WriteOutputFromProto(v.Write)
	case *sdktypes.ToolOutput_Glob:
		m.Glob = GlobOutputFromProto(v.Glob)
	case *sdktypes.ToolOutput_Grep:
		m.Grep = GrepOutputFromProto(v.Grep)
	case *sdktypes.ToolOutput_KillBash:
		m.KillBash = KillBashOutputFromProto(v.KillBash)
	case *sdktypes.ToolOutput_NotebookEdit:
		m.NotebookEdit = NotebookEditOutputFromProto(v.NotebookEdit)
	case *sdktypes.ToolOutput_WebFetch:
		m.WebFetch = WebFetchOutputFromProto(v.WebFetch)
	case *sdktypes.ToolOutput_WebSearch:
		m.WebSearch = WebSearchOutputFromProto(v.WebSearch)
	case *sdktypes.ToolOutput_TodoWrite:
		m.TodoWrite = TodoWriteOutputFromProto(v.TodoWrite)
	case *sdktypes.ToolOutput_ExitPlanMode:
		m.ExitPlanMode = ExitPlanModeOutputFromProto(v.ExitPlanMode)
	case *sdktypes.ToolOutput_ListMcpResources:
		m.ListMcpResources = ListMcpResourcesOutputFromProto(v.ListMcpResources)
	case *sdktypes.ToolOutput_ReadMcpResource:
		m.ReadMcpResource = ReadMcpResourceOutputFromProto(v.ReadMcpResource)
	case *sdktypes.ToolOutput_McpTool:
		m.McpTool = structFromProto(v.McpTool)
	}
	return m
}

func (m *ToolOutput) ToProto() *sdktypes.ToolOutput {
	if m == nil {
		return nil
	}
	pb := &sdktypes.ToolOutput{}
	switch {
	case m.Task != nil:
		pb.Output = &sdktypes.ToolOutput_Task{Task: m.Task.ToProto()}
	case m.AskUserQuestion != nil:
		pb.Output = &sdktypes.ToolOutput_AskUserQuestion{AskUserQuestion: m.AskUserQuestion.ToProto()}
	case m.Bash != nil:
		pb.Output = &sdktypes.ToolOutput_Bash{Bash: m.Bash.ToProto()}
	case m.BashOutput != nil:
		pb.Output = &sdktypes.ToolOutput_BashOutput{BashOutput: m.BashOutput.ToProto()}
	case m.Edit != nil:
		pb.Output = &sdktypes.ToolOutput_Edit{Edit: m.Edit.ToProto()}
	case m.Read != nil:
		pb.Output = &sdktypes.ToolOutput_Read{Read: m.Read.ToProto()}
	case m.Write != nil:
		pb.Output = &sdktypes.ToolOutput_Write{Write: m.Write.ToProto()}
	case m.Glob != nil:
		pb.Output = &sdktypes.ToolOutput_Glob{Glob: m.Glob.ToProto()}
	case m.Grep != nil:
		pb.Output = &sdktypes.ToolOutput_Grep{Grep: m.Grep.ToProto()}
	case m.KillBash != nil:
		pb.Output = &sdktypes.ToolOutput_KillBash{KillBash: m.KillBash.ToProto()}
	case m.NotebookEdit != nil:
		pb.Output = &sdktypes.ToolOutput_NotebookEdit{NotebookEdit: m.NotebookEdit.ToProto()}
	case m.WebFetch != nil:
		pb.Output = &sdktypes.ToolOutput_WebFetch{WebFetch: m.WebFetch.ToProto()}
	case m.WebSearch != nil:
		pb.Output = &sdktypes.ToolOutput_WebSearch{WebSearch: m.WebSearch.ToProto()}
	case m.TodoWrite != nil:
		pb.Output = &sdktypes.ToolOutput_TodoWrite{TodoWrite: m.TodoWrite.ToProto()}
	case m.ExitPlanMode != nil:
		pb.Output = &sdktypes.ToolOutput_ExitPlanMode{ExitPlanMode: m.ExitPlanMode.ToProto()}
	case m.ListMcpResources != nil:
		pb.Output = &sdktypes.ToolOutput_ListMcpResources{ListMcpResources: m.ListMcpResources.ToProto()}
	case m.ReadMcpResource != nil:
		pb.Output = &sdktypes.ToolOutput_ReadMcpResource{ReadMcpResource: m.ReadMcpResource.ToProto()}
	case m.McpTool != nil:
		pb.Output = &sdktypes.ToolOutput_McpTool{McpTool: structToProto(m.McpTool)}
	}
	return pb
}

func UsageInfoFromProto(pb *sdktypes.UsageInfo) *UsageInfo {
	if pb == nil {
		return nil
	}
	return &UsageInfo{
		InputTokens:              pb.GetInputTokens(),
		OutputTokens:             pb.GetOutputTokens(),
		CacheCreationInputTokens: optionalInt32FromProto(pb.CacheCreationInputTokens),
		CacheReadInputTokens:     optionalInt32FromProto(pb.CacheReadInputTokens),
	}
}

func (m *UsageInfo) ToProto() *sdktypes.UsageInfo {
	if m == nil {
		return nil
	}
	return &sdktypes.UsageInfo{
		InputTokens:              m.InputTokens,
		OutputTokens:             m.OutputTokens,
		CacheCreationInputTokens: optionalInt32ToProto(m.CacheCreationInputTokens),
		CacheReadInputTokens:     optionalInt32ToProto(m.CacheReadInputTokens),
	}
}

func TaskOutputFromProto(pb *sdktypes.TaskOutput) *TaskOutput {
	if pb == nil {
		return nil
	}
	return &TaskOutput{
		Result:       pb.GetResult(),
		Usage:        UsageInfoFromProto(pb.GetUsage()),
		TotalCostUsd: optionalFloat64FromProto(pb.TotalCostUsd),
		DurationMs:   optionalInt64FromProto(pb.DurationMs),
	}
}

func (m *TaskOutput) ToProto() *sdktypes.TaskOutput {
	if m == nil {
		return nil
	}
	return &sdktypes.TaskOutput{
		Result:       m.Result,
		Usage:        m.Usage.ToProto(),
		TotalCostUsd: optionalFloat64ToProto(m.TotalCostUsd),
		DurationMs:   optionalInt64ToProto(m.DurationMs),
	}
}

func AskUserQuestionOutputFromProto(pb *sdktypes.AskUserQuestionOutput) *AskUserQuestionOutput {
	if pb == nil {
		return nil
	}
	return &AskUserQuestionOutput{
		Questions: questionsFromProto(pb.GetQuestions()),
		Answers:   pb.GetAnswers(),
	}
}

func (m *AskUserQuestionOutput) ToProto() *sdktypes.AskUserQuestionOutput {
	if m == nil {
		return nil
	}
	return &sdktypes.AskUserQuestionOutput{
		Questions: questionsToProto(m.Questions),
		Answers:   m.Answers,
	}
}

func BashOutputFromProto(pb *sdktypes.BashOutput) *BashOutput {
	if pb == nil {
		return nil
	}
	return &BashOutput{
		Output:   pb.GetOutput(),
		ExitCode: pb.GetExitCode(),
		Killed:   optionalBoolFromProto(pb.Killed),
		ShellID:  optionalStringFromProto(pb.ShellId),
	}
}

func (m *BashOutput) ToProto() *sdktypes.BashOutput {
	if m == nil {
		return nil
	}
	return &sdktypes.BashOutput{
		Output:   m.Output,
		ExitCode: m.ExitCode,
		Killed:   optionalBoolToProto(m.Killed),
		ShellId:  optionalStringToProto(m.ShellID),
	}
}

func BashOutputToolOutputFromProto(pb *sdktypes.BashOutputToolOutput) *BashOutputToolOutput {
	if pb == nil {
		return nil
	}
	return &BashOutputToolOutput{
		Output:   pb.GetOutput(),
		Status:   BashOutputStatusFromProto(pb.GetStatus()),
		ExitCode: optionalInt32FromProto(pb.ExitCode),
	}
}

func (m *BashOutputToolOutput) ToProto() *sdktypes.BashOutputToolOutput {
	if m == nil {
		return nil
	}
	return &sdktypes.BashOutputToolOutput{
		Output:   m.Output,
		Status:   m.Status.ToProto(),
		ExitCode: optionalInt32ToProto(m.ExitCode),
	}
}

func EditOutputFromProto(pb *sdktypes.EditOutput) *EditOutput {
	if pb == nil {
		return nil
	}
	return &EditOutput{
		Message:      pb.GetMessage(),
		Replacements: pb.GetReplacements(),
		FilePath:     pb.GetFilePath(),
	}
}

func (m *EditOutput) ToProto() *sdktypes.EditOutput {
	if m == nil {
		return nil
	}
	return &sdktypes.EditOutput{
		Message:      m.Message,
		Replacements: m.Replacements,
		FilePath:     m.FilePath,
	}
}

func TextFileOutputFromProto(pb *sdktypes.TextFileOutput) *TextFileOutput {
	if pb == nil {
		return nil
	}
	return &TextFileOutput{
		Content:       pb.GetContent(),
		TotalLines:    pb.GetTotalLines(),
		LinesReturned: pb.GetLinesReturned(),
	}
}

func (m *TextFileOutput) ToProto() *sdktypes.TextFileOutput {
	if m == nil {
		return nil
	}
	return &sdktypes.TextFileOutput{
		Content:       m.Content,
		TotalLines:    m.TotalLines,
		LinesReturned: m.LinesReturned,
	}
}

func ImageFileOutputFromProto(pb *sdktypes.ImageFileOutput) *ImageFileOutput {
	if pb == nil {
		return nil
	}
	return &ImageFileOutput{
		Image:    pb.GetImage(),
		MimeType: pb.GetMimeType(),
		FileSize: pb.GetFileSize(),
	}
}

func (m *ImageFileOutput) ToProto() *sdktypes.ImageFileOutput {
	if m == nil {
		return nil
	}
	return &sdktypes.ImageFileOutput{
		Image:    m.Image,
		MimeType: m.MimeType,
		FileSize: m.FileSize,
	}
}

func PDFPageImageFromProto(pb *sdktypes.PDFPageImage) *PDFPageImage {
	if pb == nil {
		return nil
	}
	return &PDFPageImage{
		Image:    pb.GetImage(),
		MimeType: pb.GetMimeType(),
	}
}

func (m *PDFPageImage) ToProto() *sdktypes.PDFPageImage {
	if m == nil {
		return nil
	}
	return &sdktypes.PDFPageImage{
		Image:    m.Image,
		MimeType: m.MimeType,
	}
}

func pdfPageImagesFromProto(pbs []*sdktypes.PDFPageImage) []*PDFPageImage {
	if pbs == nil {
		return nil
	}
	result := make([]*PDFPageImage, len(pbs))
	for i, pb := range pbs {
		result[i] = PDFPageImageFromProto(pb)
	}
	return result
}

func pdfPageImagesToProto(ms []*PDFPageImage) []*sdktypes.PDFPageImage {
	if ms == nil {
		return nil
	}
	result := make([]*sdktypes.PDFPageImage, len(ms))
	for i, m := range ms {
		result[i] = m.ToProto()
	}
	return result
}

func PDFPageFromProto(pb *sdktypes.PDFPage) *PDFPage {
	if pb == nil {
		return nil
	}
	return &PDFPage{
		PageNumber: pb.GetPageNumber(),
		Text:       optionalStringFromProto(pb.Text),
		Images:     pdfPageImagesFromProto(pb.GetImages()),
	}
}

func (m *PDFPage) ToProto() *sdktypes.PDFPage {
	if m == nil {
		return nil
	}
	return &sdktypes.PDFPage{
		PageNumber: m.PageNumber,
		Text:       optionalStringToProto(m.Text),
		Images:     pdfPageImagesToProto(m.Images),
	}
}

func pdfPagesFromProto(pbs []*sdktypes.PDFPage) []*PDFPage {
	if pbs == nil {
		return nil
	}
	result := make([]*PDFPage, len(pbs))
	for i, pb := range pbs {
		result[i] = PDFPageFromProto(pb)
	}
	return result
}

func pdfPagesToProto(ms []*PDFPage) []*sdktypes.PDFPage {
	if ms == nil {
		return nil
	}
	result := make([]*sdktypes.PDFPage, len(ms))
	for i, m := range ms {
		result[i] = m.ToProto()
	}
	return result
}

func PDFFileOutputFromProto(pb *sdktypes.PDFFileOutput) *PDFFileOutput {
	if pb == nil {
		return nil
	}
	return &PDFFileOutput{
		Pages:      pdfPagesFromProto(pb.GetPages()),
		TotalPages: pb.GetTotalPages(),
	}
}

func (m *PDFFileOutput) ToProto() *sdktypes.PDFFileOutput {
	if m == nil {
		return nil
	}
	return &sdktypes.PDFFileOutput{
		Pages:      pdfPagesToProto(m.Pages),
		TotalPages: m.TotalPages,
	}
}

func NotebookCellFromProto(pb *sdktypes.NotebookCell) *NotebookCell {
	if pb == nil {
		return nil
	}
	return &NotebookCell{
		CellType:       pb.GetCellType(),
		Source:         pb.GetSource(),
		Outputs:        valuesFromProto(pb.GetOutputs()),
		ExecutionCount: optionalInt32FromProto(pb.ExecutionCount),
	}
}

func (m *NotebookCell) ToProto() *sdktypes.NotebookCell {
	if m == nil {
		return nil
	}
	return &sdktypes.NotebookCell{
		CellType:       m.CellType,
		Source:         m.Source,
		Outputs:        valuesToProto(m.Outputs),
		ExecutionCount: optionalInt32ToProto(m.ExecutionCount),
	}
}

func notebookCellsFromProto(pbs []*sdktypes.NotebookCell) []*NotebookCell {
	if pbs == nil {
		return nil
	}
	result := make([]*NotebookCell, len(pbs))
	for i, pb := range pbs {
		result[i] = NotebookCellFromProto(pb)
	}
	return result
}

func notebookCellsToProto(ms []*NotebookCell) []*sdktypes.NotebookCell {
	if ms == nil {
		return nil
	}
	result := make([]*sdktypes.NotebookCell, len(ms))
	for i, m := range ms {
		result[i] = m.ToProto()
	}
	return result
}

func NotebookFileOutputFromProto(pb *sdktypes.NotebookFileOutput) *NotebookFileOutput {
	if pb == nil {
		return nil
	}
	return &NotebookFileOutput{
		Cells:    notebookCellsFromProto(pb.GetCells()),
		Metadata: structFromProto(pb.GetMetadata()),
	}
}

func (m *NotebookFileOutput) ToProto() *sdktypes.NotebookFileOutput {
	if m == nil {
		return nil
	}
	return &sdktypes.NotebookFileOutput{
		Cells:    notebookCellsToProto(m.Cells),
		Metadata: structToProto(m.Metadata),
	}
}

func ReadOutputFromProto(pb *sdktypes.ReadOutput) *ReadOutput {
	if pb == nil {
		return nil
	}
	m := &ReadOutput{}
	switch v := pb.GetFile().(type) {
	case *sdktypes.ReadOutput_TextFile:
		m.TextFile = TextFileOutputFromProto(v.TextFile)
	case *sdktypes.ReadOutput_ImageFile:
		m.ImageFile = ImageFileOutputFromProto(v.ImageFile)
	case *sdktypes.ReadOutput_PdfFile:
		m.PdfFile = PDFFileOutputFromProto(v.PdfFile)
	case *sdktypes.ReadOutput_NotebookFile:
		m.NotebookFile = NotebookFileOutputFromProto(v.NotebookFile)
	}
	return m
}

func (m *ReadOutput) ToProto() *sdktypes.ReadOutput {
	if m == nil {
		return nil
	}
	pb := &sdktypes.ReadOutput{}
	switch {
	case m.TextFile != nil:
		pb.File = &sdktypes.ReadOutput_TextFile{TextFile: m.TextFile.ToProto()}
	case m.ImageFile != nil:
		pb.File = &sdktypes.ReadOutput_ImageFile{ImageFile: m.ImageFile.ToProto()}
	case m.PdfFile != nil:
		pb.File = &sdktypes.ReadOutput_PdfFile{PdfFile: m.PdfFile.ToProto()}
	case m.NotebookFile != nil:
		pb.File = &sdktypes.ReadOutput_NotebookFile{NotebookFile: m.NotebookFile.ToProto()}
	}
	return pb
}

func WriteOutputFromProto(pb *sdktypes.WriteOutput) *WriteOutput {
	if pb == nil {
		return nil
	}
	return &WriteOutput{
		Message:      pb.GetMessage(),
		BytesWritten: pb.GetBytesWritten(),
		FilePath:     pb.GetFilePath(),
	}
}

func (m *WriteOutput) ToProto() *sdktypes.WriteOutput {
	if m == nil {
		return nil
	}
	return &sdktypes.WriteOutput{
		Message:      m.Message,
		BytesWritten: m.BytesWritten,
		FilePath:     m.FilePath,
	}
}

func GlobOutputFromProto(pb *sdktypes.GlobOutput) *GlobOutput {
	if pb == nil {
		return nil
	}
	return &GlobOutput{
		Matches:    pb.GetMatches(),
		Count:      pb.GetCount(),
		SearchPath: pb.GetSearchPath(),
	}
}

func (m *GlobOutput) ToProto() *sdktypes.GlobOutput {
	if m == nil {
		return nil
	}
	return &sdktypes.GlobOutput{
		Matches:    m.Matches,
		Count:      m.Count,
		SearchPath: m.SearchPath,
	}
}

func GrepMatchFromProto(pb *sdktypes.GrepMatch) *GrepMatch {
	if pb == nil {
		return nil
	}
	return &GrepMatch{
		File:          pb.GetFile(),
		LineNumber:    optionalInt32FromProto(pb.LineNumber),
		Line:          pb.GetLine(),
		BeforeContext: pb.GetBeforeContext(),
		AfterContext:  pb.GetAfterContext(),
	}
}

func (m *GrepMatch) ToProto() *sdktypes.GrepMatch {
	if m == nil {
		return nil
	}
	return &sdktypes.GrepMatch{
		File:          m.File,
		LineNumber:    optionalInt32ToProto(m.LineNumber),
		Line:          m.Line,
		BeforeContext: m.BeforeContext,
		AfterContext:  m.AfterContext,
	}
}

func grepMatchesFromProto(pbs []*sdktypes.GrepMatch) []*GrepMatch {
	if pbs == nil {
		return nil
	}
	result := make([]*GrepMatch, len(pbs))
	for i, pb := range pbs {
		result[i] = GrepMatchFromProto(pb)
	}
	return result
}

func grepMatchesToProto(ms []*GrepMatch) []*sdktypes.GrepMatch {
	if ms == nil {
		return nil
	}
	result := make([]*sdktypes.GrepMatch, len(ms))
	for i, m := range ms {
		result[i] = m.ToProto()
	}
	return result
}

func GrepContentOutputFromProto(pb *sdktypes.GrepContentOutput) *GrepContentOutput {
	if pb == nil {
		return nil
	}
	return &GrepContentOutput{
		Matches:      grepMatchesFromProto(pb.GetMatches()),
		TotalMatches: pb.GetTotalMatches(),
	}
}

func (m *GrepContentOutput) ToProto() *sdktypes.GrepContentOutput {
	if m == nil {
		return nil
	}
	return &sdktypes.GrepContentOutput{
		Matches:      grepMatchesToProto(m.Matches),
		TotalMatches: m.TotalMatches,
	}
}

func GrepFilesOutputFromProto(pb *sdktypes.GrepFilesOutput) *GrepFilesOutput {
	if pb == nil {
		return nil
	}
	return &GrepFilesOutput{
		Files: pb.GetFiles(),
		Count: pb.GetCount(),
	}
}

func (m *GrepFilesOutput) ToProto() *sdktypes.GrepFilesOutput {
	if m == nil {
		return nil
	}
	return &sdktypes.GrepFilesOutput{
		Files: m.Files,
		Count: m.Count,
	}
}

func GrepFileCountFromProto(pb *sdktypes.GrepFileCount) *GrepFileCount {
	if pb == nil {
		return nil
	}
	return &GrepFileCount{
		File:  pb.GetFile(),
		Count: pb.GetCount(),
	}
}

func (m *GrepFileCount) ToProto() *sdktypes.GrepFileCount {
	if m == nil {
		return nil
	}
	return &sdktypes.GrepFileCount{
		File:  m.File,
		Count: m.Count,
	}
}

func grepFileCountsFromProto(pbs []*sdktypes.GrepFileCount) []*GrepFileCount {
	if pbs == nil {
		return nil
	}
	result := make([]*GrepFileCount, len(pbs))
	for i, pb := range pbs {
		result[i] = GrepFileCountFromProto(pb)
	}
	return result
}

func grepFileCountsToProto(ms []*GrepFileCount) []*sdktypes.GrepFileCount {
	if ms == nil {
		return nil
	}
	result := make([]*sdktypes.GrepFileCount, len(ms))
	for i, m := range ms {
		result[i] = m.ToProto()
	}
	return result
}

func GrepCountOutputFromProto(pb *sdktypes.GrepCountOutput) *GrepCountOutput {
	if pb == nil {
		return nil
	}
	return &GrepCountOutput{
		Counts: grepFileCountsFromProto(pb.GetCounts()),
		Total:  pb.GetTotal(),
	}
}

func (m *GrepCountOutput) ToProto() *sdktypes.GrepCountOutput {
	if m == nil {
		return nil
	}
	return &sdktypes.GrepCountOutput{
		Counts: grepFileCountsToProto(m.Counts),
		Total:  m.Total,
	}
}

func GrepOutputFromProto(pb *sdktypes.GrepOutput) *GrepOutput {
	if pb == nil {
		return nil
	}
	m := &GrepOutput{}
	switch v := pb.GetResult().(type) {
	case *sdktypes.GrepOutput_Content:
		m.Content = GrepContentOutputFromProto(v.Content)
	case *sdktypes.GrepOutput_Files:
		m.Files = GrepFilesOutputFromProto(v.Files)
	case *sdktypes.GrepOutput_Count:
		m.Count = GrepCountOutputFromProto(v.Count)
	}
	return m
}

func (m *GrepOutput) ToProto() *sdktypes.GrepOutput {
	if m == nil {
		return nil
	}
	pb := &sdktypes.GrepOutput{}
	switch {
	case m.Content != nil:
		pb.Result = &sdktypes.GrepOutput_Content{Content: m.Content.ToProto()}
	case m.Files != nil:
		pb.Result = &sdktypes.GrepOutput_Files{Files: m.Files.ToProto()}
	case m.Count != nil:
		pb.Result = &sdktypes.GrepOutput_Count{Count: m.Count.ToProto()}
	}
	return pb
}

func KillBashOutputFromProto(pb *sdktypes.KillBashOutput) *KillBashOutput {
	if pb == nil {
		return nil
	}
	return &KillBashOutput{
		Message: pb.GetMessage(),
		ShellID: pb.GetShellId(),
	}
}

func (m *KillBashOutput) ToProto() *sdktypes.KillBashOutput {
	if m == nil {
		return nil
	}
	return &sdktypes.KillBashOutput{
		Message: m.Message,
		ShellId: m.ShellID,
	}
}

func NotebookEditOutputFromProto(pb *sdktypes.NotebookEditOutput) *NotebookEditOutput {
	if pb == nil {
		return nil
	}
	return &NotebookEditOutput{
		Message:    pb.GetMessage(),
		EditType:   NotebookEditTypeFromProto(pb.GetEditType()),
		CellID:     optionalStringFromProto(pb.CellId),
		TotalCells: pb.GetTotalCells(),
	}
}

func (m *NotebookEditOutput) ToProto() *sdktypes.NotebookEditOutput {
	if m == nil {
		return nil
	}
	return &sdktypes.NotebookEditOutput{
		Message:    m.Message,
		EditType:   m.EditType.ToProto(),
		CellId:     optionalStringToProto(m.CellID),
		TotalCells: m.TotalCells,
	}
}

func WebFetchOutputFromProto(pb *sdktypes.WebFetchOutput) *WebFetchOutput {
	if pb == nil {
		return nil
	}
	return &WebFetchOutput{
		Response:   pb.GetResponse(),
		URL:        pb.GetUrl(),
		FinalURL:   optionalStringFromProto(pb.FinalUrl),
		StatusCode: optionalInt32FromProto(pb.StatusCode),
	}
}

func (m *WebFetchOutput) ToProto() *sdktypes.WebFetchOutput {
	if m == nil {
		return nil
	}
	return &sdktypes.WebFetchOutput{
		Response:   m.Response,
		Url:        m.URL,
		FinalUrl:   optionalStringToProto(m.FinalURL),
		StatusCode: optionalInt32ToProto(m.StatusCode),
	}
}

func WebSearchResultFromProto(pb *sdktypes.WebSearchResult) *WebSearchResult {
	if pb == nil {
		return nil
	}
	return &WebSearchResult{
		Title:    pb.GetTitle(),
		URL:      pb.GetUrl(),
		Snippet:  pb.GetSnippet(),
		Metadata: structFromProto(pb.GetMetadata()),
	}
}

func (m *WebSearchResult) ToProto() *sdktypes.WebSearchResult {
	if m == nil {
		return nil
	}
	return &sdktypes.WebSearchResult{
		Title:    m.Title,
		Url:      m.URL,
		Snippet:  m.Snippet,
		Metadata: structToProto(m.Metadata),
	}
}

func webSearchResultsFromProto(pbs []*sdktypes.WebSearchResult) []*WebSearchResult {
	if pbs == nil {
		return nil
	}
	result := make([]*WebSearchResult, len(pbs))
	for i, pb := range pbs {
		result[i] = WebSearchResultFromProto(pb)
	}
	return result
}

func webSearchResultsToProto(ms []*WebSearchResult) []*sdktypes.WebSearchResult {
	if ms == nil {
		return nil
	}
	result := make([]*sdktypes.WebSearchResult, len(ms))
	for i, m := range ms {
		result[i] = m.ToProto()
	}
	return result
}

func WebSearchOutputFromProto(pb *sdktypes.WebSearchOutput) *WebSearchOutput {
	if pb == nil {
		return nil
	}
	return &WebSearchOutput{
		Results:      webSearchResultsFromProto(pb.GetResults()),
		TotalResults: pb.GetTotalResults(),
		Query:        pb.GetQuery(),
	}
}

func (m *WebSearchOutput) ToProto() *sdktypes.WebSearchOutput {
	if m == nil {
		return nil
	}
	return &sdktypes.WebSearchOutput{
		Results:      webSearchResultsToProto(m.Results),
		TotalResults: m.TotalResults,
		Query:        m.Query,
	}
}

func TodoStatsFromProto(pb *sdktypes.TodoStats) *TodoStats {
	if pb == nil {
		return nil
	}
	return &TodoStats{
		Total:      pb.GetTotal(),
		Pending:    pb.GetPending(),
		InProgress: pb.GetInProgress(),
		Completed:  pb.GetCompleted(),
	}
}

func (m *TodoStats) ToProto() *sdktypes.TodoStats {
	if m == nil {
		return nil
	}
	return &sdktypes.TodoStats{
		Total:      m.Total,
		Pending:    m.Pending,
		InProgress: m.InProgress,
		Completed:  m.Completed,
	}
}

func TodoWriteOutputFromProto(pb *sdktypes.TodoWriteOutput) *TodoWriteOutput {
	if pb == nil {
		return nil
	}
	return &TodoWriteOutput{
		Message: pb.GetMessage(),
		Stats:   TodoStatsFromProto(pb.GetStats()),
	}
}

func (m *TodoWriteOutput) ToProto() *sdktypes.TodoWriteOutput {
	if m == nil {
		return nil
	}
	return &sdktypes.TodoWriteOutput{
		Message: m.Message,
		Stats:   m.Stats.ToProto(),
	}
}

func ExitPlanModeOutputFromProto(pb *sdktypes.ExitPlanModeOutput) *ExitPlanModeOutput {
	if pb == nil {
		return nil
	}
	return &ExitPlanModeOutput{
		Message:  pb.GetMessage(),
		Approved: optionalBoolFromProto(pb.Approved),
	}
}

func (m *ExitPlanModeOutput) ToProto() *sdktypes.ExitPlanModeOutput {
	if m == nil {
		return nil
	}
	return &sdktypes.ExitPlanModeOutput{
		Message:  m.Message,
		Approved: optionalBoolToProto(m.Approved),
	}
}

func McpResourceFromProto(pb *sdktypes.McpResource) *McpResource {
	if pb == nil {
		return nil
	}
	return &McpResource{
		URI:         pb.GetUri(),
		Name:        pb.GetName(),
		Description: optionalStringFromProto(pb.Description),
		MimeType:    optionalStringFromProto(pb.MimeType),
		Server:      pb.GetServer(),
	}
}

func (m *McpResource) ToProto() *sdktypes.McpResource {
	if m == nil {
		return nil
	}
	return &sdktypes.McpResource{
		Uri:         m.URI,
		Name:        m.Name,
		Description: optionalStringToProto(m.Description),
		MimeType:    optionalStringToProto(m.MimeType),
		Server:      m.Server,
	}
}

func mcpResourcesFromProto(pbs []*sdktypes.McpResource) []*McpResource {
	if pbs == nil {
		return nil
	}
	result := make([]*McpResource, len(pbs))
	for i, pb := range pbs {
		result[i] = McpResourceFromProto(pb)
	}
	return result
}

func mcpResourcesToProto(ms []*McpResource) []*sdktypes.McpResource {
	if ms == nil {
		return nil
	}
	result := make([]*sdktypes.McpResource, len(ms))
	for i, m := range ms {
		result[i] = m.ToProto()
	}
	return result
}

func ListMcpResourcesOutputFromProto(pb *sdktypes.ListMcpResourcesOutput) *ListMcpResourcesOutput {
	if pb == nil {
		return nil
	}
	return &ListMcpResourcesOutput{
		Resources: mcpResourcesFromProto(pb.GetResources()),
		Total:     pb.GetTotal(),
	}
}

func (m *ListMcpResourcesOutput) ToProto() *sdktypes.ListMcpResourcesOutput {
	if m == nil {
		return nil
	}
	return &sdktypes.ListMcpResourcesOutput{
		Resources: mcpResourcesToProto(m.Resources),
		Total:     m.Total,
	}
}

func McpResourceContentFromProto(pb *sdktypes.McpResourceContent) *McpResourceContent {
	if pb == nil {
		return nil
	}
	return &McpResourceContent{
		URI:      pb.GetUri(),
		MimeType: optionalStringFromProto(pb.MimeType),
		Text:     optionalStringFromProto(pb.Text),
		Blob:     optionalStringFromProto(pb.Blob),
	}
}

func (m *McpResourceContent) ToProto() *sdktypes.McpResourceContent {
	if m == nil {
		return nil
	}
	return &sdktypes.McpResourceContent{
		Uri:      m.URI,
		MimeType: optionalStringToProto(m.MimeType),
		Text:     optionalStringToProto(m.Text),
		Blob:     optionalStringToProto(m.Blob),
	}
}

func mcpResourceContentsFromProto(pbs []*sdktypes.McpResourceContent) []*McpResourceContent {
	if pbs == nil {
		return nil
	}
	result := make([]*McpResourceContent, len(pbs))
	for i, pb := range pbs {
		result[i] = McpResourceContentFromProto(pb)
	}
	return result
}

func mcpResourceContentsToProto(ms []*McpResourceContent) []*sdktypes.McpResourceContent {
	if ms == nil {
		return nil
	}
	result := make([]*sdktypes.McpResourceContent, len(ms))
	for i, m := range ms {
		result[i] = m.ToProto()
	}
	return result
}

func ReadMcpResourceOutputFromProto(pb *sdktypes.ReadMcpResourceOutput) *ReadMcpResourceOutput {
	if pb == nil {
		return nil
	}
	return &ReadMcpResourceOutput{
		Contents: mcpResourceContentsFromProto(pb.GetContents()),
		Server:   pb.GetServer(),
	}
}

func (m *ReadMcpResourceOutput) ToProto() *sdktypes.ReadMcpResourceOutput {
	if m == nil {
		return nil
	}
	return &sdktypes.ReadMcpResourceOutput{
		Contents: mcpResourceContentsToProto(m.Contents),
		Server:   m.Server,
	}
}
