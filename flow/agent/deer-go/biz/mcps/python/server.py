import asyncio
import io
import subprocess
import re
from contextlib import redirect_stdout, redirect_stderr
import traceback
from mcp.server import Server, NotificationOptions
from mcp.server.models import InitializationOptions
import mcp.server.stdio
import mcp.types as types

class PythonREPLServer:
    def __init__(self):
        self.server = Server("python-repl")
        # Shared namespace for all executions
        self.global_namespace = {
            "__builtins__": __builtins__,
        }
        print("python mcp server init success!")
        # Set up handlers using decorators
        @self.server.list_tools()
        async def handle_list_tools() -> list[types.Tool]:
            return await self.handle_list_tools()
            
        @self.server.call_tool()
        async def handle_call_tool(name: str, arguments: dict | None) -> list[types.TextContent | types.ImageContent | types.EmbeddedResource]:
            return await self.handle_call_tool(name, arguments)

    async def handle_list_tools(self) -> list[types.Tool]:
        """List available tools"""
        return [
            types.Tool(
                name="execute_python",
                description="Execute Python code and return the output. Variables persist between executions.",
                inputSchema={
                    "type": "object",
                    "properties": {
                        "code": {
                            "type": "string",
                            "description": "Python code to execute",
                        },
                        "reset": {
                            "type": "boolean",
                            "description": "Reset the Python session (clear all variables)",
                            "default": False
                        }
                    },
                    "required": ["code"],
                },
            ),
            types.Tool(
                name="list_variables",
                description="List all variables in the current session",
                inputSchema={
                    "type": "object",
                    "properties": {},
                },
            ),
            types.Tool(
                name="install_package",
                description="Install a Python package using uv",
                inputSchema={
                    "type": "object",
                    "properties": {
                        "package": {
                            "type": "string",
                            "description": "Package name to install (e.g., 'pandas')",
                        }
                    },
                    "required": ["package"],
                },
            )
        ]

    async def handle_call_tool(
        self, name: str, arguments: dict | None
    ) -> list[types.TextContent | types.ImageContent | types.EmbeddedResource]:
        """Handle tool execution requests"""
        if not arguments:
            raise ValueError("Missing arguments")
        
        if name == "execute_python":
            code = arguments.get("code")
            if not code:
                raise ValueError("Missing code parameter")

            # Check if we should reset the session
            if arguments.get("reset", False):
                self.global_namespace.clear()
                self.global_namespace["__builtins__"] = __builtins__
                return [
                    types.TextContent(
                        type="text",
                        text="Python session reset. All variables cleared."
                    )
                ]

            # Capture stdout and stderr
            stdout = io.StringIO()
            stderr = io.StringIO()
            
            try:
                # Execute code with output redirection
                with redirect_stdout(stdout), redirect_stderr(stderr):
                    exec(code, self.global_namespace)
                
                # Combine output
                output = stdout.getvalue()
                errors = stderr.getvalue()
                
                # Format response
                result = ""
                if output:
                    result += f"Output:\n{output}"
                if errors:
                    result += f"\nErrors:\n{errors}"
                if not output and not errors:
                    # Try to get the value of the last expression
                    try:
                        last_line = code.strip().split('\n')[-1]
                        last_value = eval(last_line, self.global_namespace)
                        result = f"Result: {repr(last_value)}"
                    except (SyntaxError, ValueError, NameError):
                        result = "Code executed successfully (no output)"
                
                return [
                    types.TextContent(
                        type="text",
                        text=result
                    )
                ]
                    
            except Exception as e:  # noqa: F841
                # Capture and format any exceptions
                error_msg = f"Error executing code:\n{traceback.format_exc()}"
                return [
                    types.TextContent(
                        type="text",
                        text=error_msg
                    )
                ]

        elif name == "install_package":
            package = arguments.get("package")
            if not package:
                raise ValueError("Missing package name")
                
            # Basic package name validation
            if not re.match("^[A-Za-z0-9][A-Za-z0-9._-]*$", package):
                return [
                    types.TextContent(
                        type="text",
                        text=f"Invalid package name: {package}"
                    )
                ]
            
            try:
                # Install package using uv
                process = subprocess.run(
                    ["uv", "pip", "install", package],
                    capture_output=True,
                    text=True,
                    check=True
                )

                if process.returncode != 0:
                    return [
                        types.TextContent(
                            type="text",
                            text=f"Failed to install package: {process.stderr}"
                        )
                    ]
                
                # Import the package to make it available in the REPL
                try:
                    exec(f"import {package.split('[')[0]}", self.global_namespace)
                    return [
                        types.TextContent(
                            type="text",
                            text=f"Successfully installed and imported {package}"
                        )
                    ]
                except ImportError as e:
                    return [
                        types.TextContent(
                            type="text",
                            text=f"Package installed but import failed: {str(e)}"
                        )
                    ]
                    
            except subprocess.CalledProcessError as e:
                return [
                    types.TextContent(
                        type="text",
                        text=f"Failed to install package:\n{e.stderr}"
                    )
                ]
                
        elif name == "list_variables":
            # Filter out builtins and private variables
            vars_dict = {
                k: repr(v) for k, v in self.global_namespace.items() 
                if not k.startswith('_') and k != '__builtins__'
            }
            
            if not vars_dict:
                return [
                    types.TextContent(
                        type="text",
                        text="No variables in current session."
                    )
                ]
            
            # Format variables list
            var_list = "\n".join(f"{k} = {v}" for k, v in vars_dict.items())
            return [
                types.TextContent(
                    type="text",
                    text=f"Current session variables:\n\n{var_list}"
                )
            ]
            
        else:
            raise ValueError(f"Unknown tool: {name}")

    async def run(self):
        """Run the server"""
        async with mcp.server.stdio.stdio_server() as (read_stream, write_stream):
            await self.server.run(
                read_stream,
                write_stream,
                InitializationOptions(
                    server_name="python-repl",
                    server_version="0.1.0",
                    capabilities=self.server.get_capabilities(
                        notification_options=NotificationOptions(),
                        experimental_capabilities={},
                    ),
                ),
            )

async def main():
    server = PythonREPLServer()
    await server.run()

if __name__ == "__main__":
    asyncio.run(main())
