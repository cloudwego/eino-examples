/*
 * Copyright 2025 CloudWeGo Authors
 *
 * Licensed under the Apache License, Version 2.0 (the "License");
 * you may not use this file except in compliance with the License.
 * You may obtain a copy of the License at
 *
 *     http://www.apache.org/licenses/LICENSE-2.0
 *
 * Unless required by applicable law or agreed to in writing, software
 * distributed under the License is distributed on an "AS IS" BASIS,
 * WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
 * See the License for the specific language governing permissions and
 * limitations under the License.
 */

package withoption

func ExampleHowToUse() {
	HowToUse()
	// Output:
	// === receve tool invoke ===
	// Input: {SomeInField:input value of some in field}
	// Option: &{UsingProxy:true SomeOptionField:some changed field value}
	// ===========
	// tool: {"some_out_field":"fake out value"}
	// tool_call_id: fc-xxxx
}
