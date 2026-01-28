interface CodeEditorProps {
  value: string
  onChange: (value: string) => void
}

export function CodeEditor({ value, onChange }: CodeEditorProps) {
  return (
    <div className="code-editor">
      <div className="editor-toolbar">
        <button 
          className="toolbar-btn"
          onClick={() => onChange('')}
          title="Clear"
        >
          🗑️ Clear
        </button>
        <button 
          className="toolbar-btn"
          onClick={() => navigator.clipboard.writeText(value)}
          title="Copy"
        >
          📋 Copy
        </button>
        <button 
          className="toolbar-btn"
          onClick={async () => {
            const text = await navigator.clipboard.readText()
            onChange(text)
          }}
          title="Paste"
        >
          📥 Paste
        </button>
      </div>
      <textarea
        className="code-textarea"
        value={value}
        onChange={(e) => onChange(e.target.value)}
        spellCheck={false}
        placeholder="Paste your Terraform or Bicep code here..."
      />
      <div className="editor-footer">
        <span className="line-count">
          {value.split('\n').length} lines
        </span>
        <span className="file-type">
          {value.includes('resource "azurerm_') ? 'Terraform' : 
           value.includes("param ") ? 'Bicep' : 'Auto-detect'}
        </span>
      </div>
    </div>
  )
}
