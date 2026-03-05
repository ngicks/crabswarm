package models

import "encoding/json"

type ToolOutputSchemas interface {
	toolOutputSchemas()
	json.Marshaler
	json.Unmarshaler
}

// As per https://platform.claude.com/docs/en/agent-sdk/typescript#tool-output-schemas
/*
type ToolOutputSchemas =
  | AgentOutput
  | AskUserQuestionOutput
  | BashOutput
  | ConfigOutput
  | EnterWorktreeOutput
  | ExitPlanModeOutput
  | FileEditOutput
  | FileReadOutput
  | FileWriteOutput
  | GlobOutput
  | GrepOutput
  | ListMcpResourcesOutput
  | NotebookEditOutput
  | ReadMcpResourceOutput
  | TaskStopOutput
  | TodoWriteOutput
  | WebFetchOutput
  | WebSearchOutput;
*/

// fallback target
func (*UnknownOutput) toolOutputSchemas()        {}

func (*AgentOutput) toolOutputSchemas()            {}
func (*AskUserQuestionOutput) toolOutputSchemas()  {}
func (*BashOutput) toolOutputSchemas()             {}
func (*ConfigOutput) toolOutputSchemas()           {}
func (*EnterWorktreeOutput) toolOutputSchemas()    {}
func (*ExitPlanModeOutput) toolOutputSchemas()     {}
func (*FileEditOutput) toolOutputSchemas()         {}
func (*FileReadOutput) toolOutputSchemas()         {}
func (*FileWriteOutput) toolOutputSchemas()        {}
func (*GlobOutput) toolOutputSchemas()             {}
func (*GrepOutput) toolOutputSchemas()             {}
func (*ListMcpResourcesOutput) toolOutputSchemas() {}
func (*NotebookEditOutput) toolOutputSchemas()     {}
func (*ReadMcpResourceOutput) toolOutputSchemas()  {}
func (*TaskStopOutput) toolOutputSchemas()         {}
func (*TodoWriteOutput) toolOutputSchemas()        {}
func (*WebFetchOutput) toolOutputSchemas()         {}
func (*WebSearchOutput) toolOutputSchemas()        {}



func unmarshalToolOutputSchemas(toolName string, data []byte) (_ ToolOutputSchemas, err error) {
	defer func() {
		if err != nil {
			err = fmt.Errorf("ToolOutputSchemas: %w", err)
		}
	}()

	switch toolName {
	default:
		// unmarshal to default
			case "":
		return nil, fmt.Errorf("empty tool_name")
	case "Task":
     v, err :=		unmarshalAgentOutput(data)
		 if err != nil {
		 return nil, err
		 }
		 return v, nil
		case "AskUserQuestion":
			var v AskUserQuestionOutput 
			err := json.Unmarshal(data, &v)
			return v, err
		case "Bash":
						var v BashOutput 
			err := json.Unmarshal(data, &v)
			return v, err
		case "Edit":
			var v FileEditOutput 
						err := json.Unmarshal(data, &v)
			return v, err
		case "Read":
			v, err := unmarshalFileReadOutput(data)
			if err != nil {
return nil, err
			}
			return v, nil
		case "Write":
			var v FileWriteOutput
						err := json.Unmarshal(data, &v)
			return v, err
		case  "Glob":
			var v GlobOutput 
						err := json.Unmarshal(data, &v)
			return v, err
		case "Grep":	
			var v GrepOutput  
						err := json.Unmarshal(data, &v)
			return v, err
		case "TaskStop":
			var v TaskStopOutput 
						err := json.Unmarshal(data, &v)
			return v, err
		case "NotebookEdit":
			var v NotebookEditOutput
						err := json.Unmarshal(data, &v)
			return v, err
		case "WebFetch":
			var v WebFetchOutput
						err := json.Unmarshal(data, &v)
			return v, err
		case "WebSearch":
			var v WebSearchOutput 
						err := json.Unmarshal(data, &v)
			return v, err
		case "TodoWrite":
			var v TodoWriteOutput  
						err := json.Unmarshal(data, &v)
			return v, err
		case "ExitPlanMode":
			var v ExitPlanModeOutput 
						err := json.Unmarshal(data, &v)
			return v, err
		case "ListMcpResources":
			var v ListMcpResourcesOutput	
						err := json.Unmarshal(data, &v)
			return v, err
		case "ReadMcpResource":
			var v ReadMcpResourceOutput 
						err := json.Unmarshal(data, &v)
			return v, err
		case "Config":
			var v ConfigOutput 
						err := json.Unmarshal(data, &v)
			return v, err
		case "EnterWorktree":
			 var v EnterWorktreeOutput 
						err := json.Unmarshal(data, &v)
			return v, err
	}
}

// AgentOutput is direct translation of https://platform.claude.com/docs/en/agent-sdk/typescript#task-2
// Task name: "Task".
type AgentOutput interface {
	agentOutput()
}

func(AgentOutputCompleted) agentOutput(){}
func(AgentOutputAsyncLaunched) agentOutput(){}
func(AgentOutputSubAgentEntered) agentOutput(){}

func unmarshalAgentOutput(data []byte) (_ AgentOutput, err error) {}

type AgentOutputCompleted struct{
      status: "completed";
      agentId: string;
      content: Array<{ type: "text"; text: string }>;
      totalToolUseCount: number;
      totalDurationMs: number;
      totalTokens: number;
      usage: {
        input_tokens: number;
        output_tokens: number;
        cache_creation_input_tokens: number | null;
        cache_read_input_tokens: number | null;
        server_tool_use: {
          web_search_requests: number;
          web_fetch_requests: number;
        } | null;
        service_tier: ("standard" | "priority" | "batch") | null;
        cache_creation: {
          ephemeral_1h_input_tokens: number;
          ephemeral_5m_input_tokens: number;
        } | null;
      };
      prompt: string;
}

type AgentOutputAsyncLaunched struct{
	      status: "async_launched";
      agentId: string;
      description: string;
      prompt: string;
      outputFile: string;
      canReadOutputFile?: boolean;
}

type AgentOutputSubAgentEntered  struct{
      status: "sub_agent_entered";
      description: string;
      message: string;
}

// AskUserQuestionOutput is direct translation of https://platform.claude.com/docs/en/agent-sdk/typescript#ask-user-question-2
// Tool name: AskUserQuestion
type AskUserQuestionOutput struct{
	Questions  []AskUserQuestionOutputQuestion  `json:"questions"`
  answers: Record<string, string>;
}

type AskUserQuestionOutputQuestion struct{
	  question: string;
    header: string;
    options: []AskUserQuestionOutputQuestionOption
    multiSelect: boolean;
}

type  AskUserQuestionOutputQuestionOption struct{
label: string
description: string
}


// BashOutput is direct translation of https://platform.claude.com/docs/en/agent-sdk/typescript#ask-user-question-2
// Tool name: Bash
type BashOutput struct{
  stdout: string;
  stderr: string;
  rawOutputPath?: string;
  interrupted: boolean;
  isImage?: boolean;
  backgroundTaskId?: string;
  backgroundedByUser?: boolean;
  dangerouslyDisableSandbox?: boolean;
  returnCodeInterpretation?: string;
  structuredContent?: unknown[];
  persistedOutputPath?: string;
  persistedOutputSize?: number;
}

// FileEditOutput is direct translation of https://platform.claude.com/docs/en/agent-sdk/typescript#edit-2
// Tool name: Edit
type FileEditOutput struct{
  filePath: string;
  oldString: string;
  newString: string;
  originalFile: string;
  structuredPatch: Array<{
    oldStart: number;
    oldLines: number;
    newStart: number;
    newLines: number;
    lines: string[];
  }>;
  userModified: boolean;
  replaceAll: boolean;
  gitDiff?: {
    filename: string;
    status: "modified" | "added";
    additions: number;
    deletions: number;
    changes: number;
    patch: string;
  };
}

// FileReadOutput is direct translation of https://platform.claude.com/docs/en/agent-sdk/typescript#edit-2
// Tool name: Read
type FileReadOutput interface{
	fileReadOutput()
}

func unmarshalFileReadOutput(data []byte) (_ FileReadOutput, err error) {}


type FileReadOutputText struct{
	     type: "text";
      file: {
        filePath: string;
        content: string;
        numLines: number;
        startLine: number;
        totalLines: number;
      };
}

type FileReadOutputImage struct{
	      type: "image";
      file: {
        base64: string;
        type: "image/jpeg" | "image/png" | "image/gif" | "image/webp";
        originalSize: number;
        dimensions?: {
          originalWidth?: number;
          originalHeight?: number;
          displayWidth?: number;
          displayHeight?: number;
        };
      };
}

type FileReadOutputNotebook struct{
     type: "notebook";
      file: {
        filePath: string;
        cells: unknown[];
      };
}

type FileReadOutputPdf struct{
	      type: "pdf";
      file: {
        filePath: string;
        base64: string;
        originalSize: number;
      };
}

type FileReadOutputParts struct{
	      type: "parts";
      file: {
        filePath: string;
        originalSize: number;
        count: number;
        outputDir: string;
      };
}

// FileWriteOutput is direct translation of https://platform.claude.com/docs/en/agent-sdk/typescript#write-2
// Tool name: Write
type FileWriteOutput struct{
  type: "create" | "update";
  filePath: string;
  content: string;
  structuredPatch: Array<{
    oldStart: number;
    oldLines: number;
    newStart: number;
    newLines: number;
    lines: string[];
  }>;
  originalFile: string | null;
  gitDiff?: {
    filename: string;
    status: "modified" | "added";
    additions: number;
    deletions: number;
    changes: number;
    patch: string;
  };
}


// GlobOutput is direct translation of https://platform.claude.com/docs/en/agent-sdk/typescript#glob-2
// Tool name: Glob
type GlobOutput struct{
  durationMs: number;
  numFiles: number;
  filenames: string[];
  truncated: boolean;
}

// GrepOutput is direct translation of https://platform.claude.com/docs/en/agent-sdk/typescript#grep-2
// Tool name: Grep
type GrepOutput struct{
  mode?: "content" | "files_with_matches" | "count";
  numFiles: number;
  filenames: string[];
  content?: string;
  numLines?: number;
  numMatches?: number;
  appliedLimit?: number;
  appliedOffset?: number;
}

// TaskStopOutput is direct translation of https://platform.claude.com/docs/en/agent-sdk/typescript#task-stop-2
// Tool name: TaskStop
type TaskStopOutput struct{
  message: string;
  task_id: string;
  task_type: string;
  command?: string;
}

// NotebookEditOutput is direct translation of https://platform.claude.com/docs/en/agent-sdk/typescript#notebook-edit-2
// Tool name: NotebookEdit
type NotebookEditOutput struct{
  new_source: string;
  cell_id?: string;
  cell_type: "code" | "markdown";
  language: string;
  edit_mode: string;
  error?: string;
  notebook_path: string;
  original_file: string;
  updated_file: string;
}

// WebFetchOutput is direct translation of https://platform.claude.com/docs/en/agent-sdk/typescript#web-fetch-2
// Tool name: WebFetch
type WebFetchOutput struct{
  bytes: number;
  code: number;
  codeText: string;
  result: string;
  durationMs: number;
  url: string;
}

// WebSearchOutput is direct translation of https://platform.claude.com/docs/en/agent-sdk/typescript#web-search-2
// Tool name: WebSearch
type WebSearchOutput struct{
  query: string;
  results: Array<
    | {
        tool_use_id: string;
        content: Array<{ title: string; url: string }>;
      }
    | string
  >;
  durationSeconds: number;
}

// TodoWriteOutput is direct translation of https://platform.claude.com/docs/en/agent-sdk/typescript#todo-write-2
// Tool name: TodoWrite
type TodoWriteOutput struct{
  oldTodos: Array<{
    content: string;
    status: "pending" | "in_progress" | "completed";
    activeForm: string;
  }>;
  newTodos: Array<{
    content: string;
    status: "pending" | "in_progress" | "completed";
    activeForm: string;
  }>;
}

// ExitPlanModeOutput is direct translation of https://platform.claude.com/docs/en/agent-sdk/typescript#exit-plan-mode-2
// Tool name: ExitPlanMode
type ExitPlanModeOutput struct{
  plan: string | null;
  isAgent: boolean;
  filePath?: string;
  hasTaskTool?: boolean;
  awaitingLeaderApproval?: boolean;
  requestId?: string;
}

// ListMcpResourcesOutput is direct translation of https://platform.claude.com/docs/en/agent-sdk/typescript#list-mcp-resources-2
// Tool name: ListMcpResources
type ListMcpResourcesOutput  []ListMcpResourcesOutputSingle

type ListMcpResourcesOutputSingle  struct{
  uri: string;
  name: string;
  mimeType?: string;
  description?: string;
  server: string;
}

// ReadMcpResourceOutput is direct translation of https://platform.claude.com/docs/en/agent-sdk/typescript#read-mcp-resource-2
// Tool name: ReadMcpResource
type ReadMcpResourceOutput struct{
  contents: Array<{
    uri: string;
    mimeType?: string;
    text?: string;
  }>;
}

// ConfigOutput is direct translation of https://platform.claude.com/docs/en/agent-sdk/typescript#config-2
// Tool name: Config
type ConfigOutput struct{
	  success: boolean;
  operation?: "get" | "set";
  setting?: string;
  value?: unknown;
  previousValue?: unknown;
  newValue?: unknown;
  error?: string;
}

// EnterWorktreeOutput is direct translation of https://platform.claude.com/docs/en/agent-sdk/typescript#enter-worktree-2
// Tool name: EnterWorktree
type EnterWorktreeOutput struct{
  worktreePath: string;
  worktreeBranch?: string;
  message: string;
}









