import { describe, expect, it } from "vitest"
import { buildCurlCommand, type CurlRequest } from "./curlCommand"

function request(partial: Partial<CurlRequest> = {}): CurlRequest {
  return {
    id: "request-1",
    name: "Variable request",
    method: "GET",
    url: "https://example.com",
    params: [],
    headers: [],
    body: { type: "none", raw: "", json: "", formUrlEncoded: [], formData: [] },
    auth: {
      type: "none",
      basic: { username: "", password: "" },
      bearer: { token: "" },
      apiKey: { key: "", value: "", addTo: "header" },
    },
    ...partial,
  }
}

describe("buildCurlCommand", () => {
  it("resolves active environment variables before encoding request fields", () => {
    const command = buildCurlCommand(request({
      method: "POST",
      url: "{{baseUrl}}/users",
      params: [{ key: "{{queryKey}}", value: "{{queryValue}}", description: "" }],
      headers: [{ key: "X-Tenant", value: "{{tenant}}", description: "" }],
      body: {
        type: "json",
        raw: "",
        json: "{\"name\":\"{{name}}\"}",
        formUrlEncoded: [],
        formData: [],
      },
      auth: {
        type: "bearer",
        basic: { username: "", password: "" },
        bearer: { token: "{{token}}" },
        apiKey: { key: "", value: "", addTo: "header" },
      },
    }), [
      { key: "baseUrl", value: "https://api.example.com" },
      { key: "queryKey", value: "search term" },
      { key: "queryValue", value: "O'Reilly" },
      { key: "tenant", value: "tenant-a" },
      { key: "name", value: "D'Angelo" },
      { key: "token", value: "secret-token" },
    ])

    expect(command).toContain(["'https://api.example.com/users?search%20term=O", "'\"'\"'", "Reilly'"].join(""))
    expect(command).toContain("-H 'X-Tenant: tenant-a'")
    expect(command).toContain("-H 'Authorization: Bearer secret-token'")
    expect(command).toContain(`-d '{"name":"D'"'"'Angelo"}'`)
    expect(command).not.toContain("{{")
  })

  it("exports query API keys and keeps unresolved variables visible", () => {
    const command = buildCurlCommand(request({
      url: "api.example.com/{{version}}",
      auth: {
        type: "api-key",
        basic: { username: "", password: "" },
        bearer: { token: "" },
        apiKey: { key: "{{apiKeyName}}", value: "{{apiKeyValue}}", addTo: "query" },
      },
    }), [
      { key: "apiKeyName", value: "access key" },
      { key: "apiKeyValue", value: "abc/123" },
    ])

    expect(command).toContain("'api.example.com/{{version}}?access%20key=abc%2F123'")
  })

  it("exports basic auth and URL-encoded forms with resolved values", () => {
    const command = buildCurlCommand(request({
      method: "POST",
      body: {
        type: "form-urlencoded",
        raw: "",
        json: "",
        formUrlEncoded: [{ key: "{{field}}", value: "{{value}}", description: "" }],
        formData: [],
      },
      auth: {
        type: "basic",
        basic: { username: "{{username}}", password: "{{password}}" },
        bearer: { token: "" },
        apiKey: { key: "", value: "", addTo: "header" },
      },
    }), [
      { key: "field", value: "display name" },
      { key: "value", value: "Mini Post" },
      { key: "username", value: "api-user" },
      { key: "password", value: "p'ass" },
    ])

    expect(command).toContain(`-u 'api-user:p'"'"'ass'`)
    expect(command).toContain("-d 'display%20name=Mini%20Post'")
  })

  it("exports text and file multipart fields with resolved values", () => {
    const command = buildCurlCommand(request({
      method: "POST",
      body: {
        type: "form-data",
        raw: "",
        json: "",
        formUrlEncoded: [],
        formData: [
          { key: "description", value: "{{description}}", description: "", type: "text", filePath: "", fileName: "" },
          { key: "{{fileField}}", value: "", description: "", type: "file", filePath: "{{filePath}}", fileName: "data.json" },
          { key: "empty-file", value: "", description: "", type: "file", filePath: "", fileName: "" },
        ],
      },
    }), [
      { key: "description", value: "release artifact" },
      { key: "fileField", value: "payload" },
      { key: "filePath", value: "/tmp/release data.json" },
    ])

    expect(command).toContain("--form-string 'description=release artifact'")
    expect(command).toContain("-F 'payload=@/tmp/release data.json'")
    expect(command).not.toContain("empty-file")
  })

  it("exports header API keys after variable resolution", () => {
    const command = buildCurlCommand(request({
      auth: {
        type: "api-key",
        basic: { username: "", password: "" },
        bearer: { token: "" },
        apiKey: { key: "{{headerName}}", value: "{{headerValue}}", addTo: "header" },
      },
    }), [
      { key: "headerName", value: "X-API-Key" },
      { key: "headerValue", value: "secret" },
    ])

    expect(command).toContain("-H 'X-API-Key: secret'")
  })

  it("omits disabled editor rows from the copied command", () => {
    const command = buildCurlCommand(request({
      method: "POST",
      params: [
        { key: "included", value: "yes", enabled: true },
        { key: "disabled-param", value: "no", enabled: false },
      ],
      headers: [
        { key: "X-Included", value: "yes", enabled: true },
        { key: "X-Disabled", value: "no", enabled: false },
      ],
      body: {
        type: "form-data",
        formData: [
          { key: "included-field", value: "yes", type: "text", enabled: true },
          { key: "disabled-field", value: "no", type: "text", enabled: false },
        ],
      },
    }))

    expect(command).toContain("included=yes")
    expect(command).toContain("X-Included: yes")
    expect(command).toContain("included-field=yes")
    expect(command).not.toContain("disabled-param")
    expect(command).not.toContain("X-Disabled")
    expect(command).not.toContain("disabled-field")
  })
})
