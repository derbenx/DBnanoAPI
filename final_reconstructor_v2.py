import re

def strip_strings_and_comments(code):
    code = re.sub(r'/\*.*?\*/', ' ', code, flags=re.DOTALL)
    code = re.sub(r';.*$', '', code, flags=re.MULTILINE)
    code = re.sub(r'"([^"]|"")*"', '""', code)
    code = re.sub(r"'([^']|'')*'", "''", code)
    return code

with open('nanoV3.ahk', 'r', encoding='utf-8', errors='ignore') as f:
    content = f.read()

# Separate global code
pattern = re.compile(r'^([a-zA-Z0-9_]+)\(.*?\)\s*\{', re.MULTILINE)
matches = list(pattern.finditer(content))

global_code = content[:matches[0].start()]

# Function map to store the last version of each function
func_map = {}

for i, match in enumerate(matches):
    name = match.group(1)
    start = match.start()
    if i + 1 < len(matches):
        end = matches[i+1].start()
    else:
        end = len(content)

    body = content[start:end].rstrip()

    # Try to find balanced end within this segment
    clean_body = strip_strings_and_comments(body)
    count = 0
    real_end = -1
    for j, char in enumerate(clean_body):
        if char == '{':
            count += 1
        elif char == '}':
            count -= 1
            if count == 0:
                real_end = j
                break

    # If unclosed, it will be closed later.
    # If over-closed, we'll fix it.

    func_map[name] = body

# Now rebuild. We'll use a fixed order or the order found.
seen = set()
ordered_funcs = []
for match in matches:
    name = match.group(1)
    if name not in seen:
        seen.add(name)
        # For each function, we want the LAST version found in the file
        last_body = ""
        for m in reversed(matches):
            if m.group(1) == name:
                start = m.start()
                # Find next function or end
                nxt = content.find('\n', start) # Start of next line
                # Actually just use the logic from above but find the LAST version
                idx = matches.index(m)
                start = matches[idx].start()
                if idx + 1 < len(matches):
                    end = matches[idx+1].start()
                else:
                    end = len(content)
                last_body = content[start:end].rstrip()
                break

        # Balance braces for this body
        clean = strip_strings_and_comments(last_body)
        diff = clean.count('{') - clean.count('}')
        if diff > 0:
            last_body += '\n' + ('}' * diff)
        elif diff < 0:
            for _ in range(-diff):
                last_brace = last_body.rfind('}')
                if last_brace != -1:
                    last_body = last_body[:last_brace] + last_body[last_brace+1:]

        ordered_funcs.append(last_body)

# Hot fix for specific known issues if balancing didn't work perfectly
final_content = global_code + '\n\n' + '\n\n'.join(ordered_funcs)

# Manual fix for the reported "Unexpected Else" in ShowTaskForm
# It's caused by '}' being in the middle of if/else
final_content = re.sub(r'currentImgPath := LV_Images\.GetText\(focusedRow, 5\)\s*\}',
                       r'currentImgPath := LV_Images.GetText(focusedRow, 5)', final_content)

with open('nanoV3.ahk', 'w', encoding='utf-8') as f:
    f.write(final_content)
