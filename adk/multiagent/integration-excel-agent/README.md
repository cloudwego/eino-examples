## Eino ADK Integration Example: Excel Agent

### Description
Excel Agent is a "smart assistant who can understand Excel". It first breaks down the problem into steps, then executes and verifies the results step by step. It can understand user problems and uploaded file contents, propose feasible solutions, and select appropriate tools (system commands, generate and run Python code, network queries, etc.) to complete tasks.

Before using this, you need to make some configurations:

### Env
```
//（required）Basic LLM Model Config，currently support Ark and OpenAI models.
// Ark model config
export ARK_API_KEY=""   //（required）Ark Model API Key
export ARK_MODEL=""     //（required）Ark Model name
export ARK_BASE_URL=""  //（optional）Ark Model base_url
export ARK_REGION=""    //（optional）Ark Model region

// OpenAI model config
export OPENAI_API_KEY=""       // (required）OpenAI Model API Key
export OPENAI_MODEL=""         // (required）OpenAI Model name
export OPENAI_BASE_URL=""      // (optional）OpenAI base_url
export OPENAI_BY_AZURE="false" // (optional）OpenAI using Azure service or not

//（optional）Python executable path，default using system python.
// It's recommended to use venv, and install pandas / numpy / matplotlib / openpyxl before lanunching this agent.
// When the code written by CodeAgent fails to run due to lack of dependencies, the pip command may be used to try to install dependencies. 
// At this time, if you use python in the system environment, it may be blocked, causing the Excel Agent to fail to run and exit.
export EXCEL_AGENT_PYTHON_EXECUTABLE_PATH="python"

//（optional）Vision Model Config，default null, which ImagepReader tool in ReportAgent will not activate.
export ARK_VISION_API_KEY=""    // Ark Vision Model API Key
export ARK_VISION_MODEL=""      // Ark Vision Model name
export ARK_VISION_BASE_URL=""   // Ark Vision Model base_url
export ARK_VISION_REGION=""     // Ark Vision Model region
```

### Input 
The input for Excel Agent is a description of user requirements and a series of files to be processed:
- The first line in `main.go` represents the requirement description entered by the user:
  ```
    func main() {
        // query := schema.UserMessage("Count the recommended novel names and recommended times in the attachment file, and write the results to the file. The content with "" is the name of the novel and forms a table. The header is the name of the novel and the number of recommendations. The novels with the same name are listed in only one line, and the number of recommendations is added")
        // query := schema.UserMessage("Read the content in simulated question. csv, put the question, answer, resolution and options in the same line in a standardized format, and simply write the answer to the resolution")
        query := schema.UserMessage("Please help me extract the first column in question.csv table into a new csv")
    }
  ```
- `adk/multiagent/integration-excel-agent/playground/input` is the default attachment input path. For example, the `question.csv` file mentioned in the above query needs to be placed in this directory before it can be read by the agent. In addition, it supports the configuration of the environment variable `EXCEL_AGENT_INPUT_DIR` to set the attachment input path (absolute path).
- Several sample files are provided in the path `adk/multiagent/integration-excel-agent/playground/test_data` for your test:
  ```
    % tree adk/multiagent/integration-excel-agent/playground/test_data
    adk/multiagent/integration-excel-agent/playground/test_data
    ├── questions.csv
    ├── 推荐小说.txt
    └── 模拟出题.csv
    
    1 directory, 3 files
  ```

### Output
The default working directory is `adk/multiagent/integration-excel-agent/playground/${uuid}`. 

You can set your own working directory by setting env: `export EXCEL_AGENT_WORK_DIR="your_path""` (the absolute path before/$uuid).