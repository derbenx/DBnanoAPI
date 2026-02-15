import re

def strip_strings_and_comments(code):
    code = re.sub(r'/\*.*?\*/', ' ', code, flags=re.DOTALL)
    code = re.sub(r';.*$', '', code, flags=re.MULTILINE)
    code = re.sub(r'"([^"]|"")*"', '""', code)
    code = re.sub(r"'([^']|'')*'", "''", code)
    return code

with open('nanoV3.ahk', 'r', encoding='utf-8', errors='ignore') as f:
    lines = f.readlines()

for i, line in enumerate(lines, 1):
    stripped = line.strip()
    if stripped.startswith('else') or stripped.startswith('catch') or stripped.startswith('finally'):
        # Check previous non-empty line
        prev_idx = i - 2
        while prev_idx >= 0 and not lines[prev_idx].strip():
            prev_idx -= 1

        if prev_idx >= 0:
            prev_line = lines[prev_idx].strip()
            # In AHK v2, else/catch must follow a brace } or a single-line statement.
            # But usually they follow a brace.
            # If the user is getting "Unexpected Else", it means the brace closed the block entirely.

            # Let's check for braces on the current line or previous line
            if not stripped.startswith('}') and not prev_line.endswith('{'):
                 # This is fine if it's a single line if, but we saw an error.
                 pass

# A better check: verify each function body for brace balance.
pattern = re.compile(r'^([a-zA-Z0-9_]+)\(.*?\)\s*\{', re.MULTILINE)
matches = list(pattern.finditer("".join(lines)))

for i, match in enumerate(matches):
    start = match.start()
    name = match.group(1)
    if i + 1 < len(matches):
        end = matches[i+1].start()
    else:
        end = len("".join(lines))

    func_content = "".join(lines)[start:end]
    clean = strip_strings_and_comments(func_content)

    # We need to find the REAL end of the function by counting braces.
    count = 0
    real_end = -1
    for j, char in enumerate(clean):
        if char == '{':
            count += 1
        elif char == '}':
            count -= 1
            if count == 0:
                real_end = j
                break

    if real_end == -1:
        print(f"Function {name} at line {i} is unclosed!")
    else:
        # Check if there is extra stuff after the brace but before the next function
        remaining = clean[real_end+1:].strip()
        if remaining:
             print(f"Function {name} has extra characters after closing brace: {remaining[:50]}")
