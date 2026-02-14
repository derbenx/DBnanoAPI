import os
import re

with open('nanoV3.ahk', 'r', encoding='utf-8') as f:
    lines = f.readlines()

# Ensure we have the original file content
if len(lines) < 1000:
    print("Warning: File seems too short, might already be corrupted.")

# 1. Add useCurl global and check
for i in range(len(lines)):
    if 'global API_KEY :=' in lines[i]:
        lines.insert(i, 'global useCurl := 0\n')
        lines.insert(i+1, 'if (useCurl && !FileExist(A_ScriptDir . "\\curl.exe"))\n')
        lines.insert(i+2, '    useCurl := 0\n')
        break

# 2. Add other globals
for i in range(len(lines)):
    if 'global NextImageID := 1' in lines[i]:
        lines.insert(i+1, 'global PendingTasks := 0\n')
        lines.insert(i+2, 'global CurlTimers := Map()\n')
        break

content = "".join(lines)

# 3. Update RunGeminiTask for curl non-blocking
# First, update global list in function
old_rg_globals = 'global API_KEY, MODEL1, MODEL2, hurl, encourage'
new_rg_globals = 'global API_KEY, MODEL1, MODEL2, hurl, encourage, useCurl, PendingTasks, CurlTimers'
content = content.replace(old_rg_globals, new_rg_globals)

# Insert curl logic into RunGeminiTask
curl_logic = '''
    MODEL_ID := InStr(agent, "Flash") ? MODEL1 : MODEL2
    if (useCurl) {
        payload := CreateJsonPayload(taskObj, fullPath)
        payloadFile := A_Temp . "\\\\gemini_pay_" . A_TickCount . "_" . batchIdx . ".json"
        responseFile := A_Temp . "\\\\gemini_res_" . A_TickCount . "_" . batchIdx . ".json"
        if FileExist(payloadFile)
            FileDelete(payloadFile)
        FileAppend(payload, payloadFile, "UTF-8-RAW")
        apiUrl := hurl . MODEL_ID . ":streamGenerateContent?key=" . API_KEY
        curlCmd := 'curl -s -N -X POST "' . apiUrl . '" -H "Content-Type: application/json" -d "@' . payloadFile . '" -o "' . responseFile . '"'
        Run(curlCmd, , "Hide", &pid)
        global PendingTasks += 1
        CurlTimers[pid] := CheckCurlProgress.Bind(pid, responseFile, payloadFile, batchIdx, nameNoExt)
        SetTimer(CurlTimers[pid], 200)
        ModelLog.Value .= "`n[curl] Task " . batchIdx . " started (PID: " . pid . ")"
        SendMessage(0x0115, 7, 0, ModelLog.Hwnd, "A")
        return
    }
'''
content = content.replace('    MODEL_ID := InStr(agent, "Flash") ? MODEL1 : MODEL2', curl_logic)

# 4. Update CheckBatchStatus for curl
old_cbs = '''CheckBatchStatus(jobID, targetRow) {
    whr := ComObject("WinHttp.WinHttpRequest.5.1")'''
new_cbs = '''CheckBatchStatus(jobID, targetRow) {
    global useCurl, API_KEY
    if (useCurl) {
        url := "https://generativelanguage.googleapis.com/v1beta/" . jobID . "?key=" . API_KEY
        resFile := A_Temp . "\\\\gemini_status_" . A_TickCount . ".json"
        curlCmd := 'curl -s "' . url . '" -o "' . resFile . '"'
        RunWait(curlCmd, , "Hide")
        resText := FileRead(resFile)
        FileDelete(resFile)
        state := JSON_Get(resText, "state")
        if (state == "SUCCEEDED" || state == "BATCH_STATE_SUCCEEDED") {
            outputUri := JSON_Get(resText, "responsesFile")
            return outputUri
        }
        return ""
    }
    whr := ComObject("WinHttp.WinHttpRequest.5.1")'''
content = content.replace(old_cbs, new_cbs)

# 5. Update UploadBatchFile for curl
content = content.replace('UploadBatchFile(FilePath) {', 'UploadBatchFile(FilePath) {\n    global useCurl, API_KEY')
upload_curl = '''
    offset += fileData.Size
    StrPut(bodyEnd, combinedBody.Ptr + offset, "UTF-8")

    if (useCurl) {
        tempBodyFile := A_Temp . "\\\\gemini_upload_" . A_TickCount . ".bin"
        resFile := A_Temp . "\\\\gemini_upload_res_" . A_TickCount . ".json"
        FileOpen(tempBodyFile, "w", "cp0").RawWrite(combinedBody)
        url := "https://generativelanguage.googleapis.com/upload/v1beta/files?key=" . API_KEY
        curlCmd := 'curl -s -X POST "' . url . '" -H "X-Goog-Upload-Protocol: multipart" -H "Content-Type: multipart/related; boundary=' . boundary . '" --data-binary "@' . tempBodyFile . '" -o "' . resFile . '"'
        RunWait(curlCmd, , "Hide")
        resText := FileRead(resFile)
        try FileDelete(tempBodyFile)
        try FileDelete(resFile)
        if RegExMatch(resText, '"uri":\\s*"([^"]+)"', &match)
            return match[1]
        throw Error("Curl Upload Failed: " . resText)
    }
'''
content = content.replace('    offset += fileData.Size\n    StrPut(bodyEnd, combinedBody.Ptr + offset, "UTF-8")', upload_curl)

# 6. Update DownloadAndSaveBatch for curl
content = content.replace('DownloadAndSaveBatch(outputUri) {', 'DownloadAndSaveBatch(outputUri) {\n    global useCurl, API_KEY')
download_curl = '''
    finalUrl := "https://generativelanguage.googleapis.com/v1beta/" . outputUri . ":download?alt=media&key=" . API_KEY

    rawResponse := ""
    if (useCurl) {
        resFile := A_Temp . "\\\\gemini_batch_res_" . A_TickCount . ".jsonl"
        curlCmd := 'curl -s "' . finalUrl . '" -o "' . resFile . '"'
        RunWait(curlCmd, , "Hide")
        if FileExist(resFile) {
            rawResponse := FileRead(resFile)
            FileDelete(resFile)
        }
    } else {
        whr := ComObject("WinHttp.WinHttpRequest.5.1")
        whr.SetTimeouts(30000, 60000, 600000, 600000)
        try {
            whr.Open("GET", finalUrl, false)
            whr.Send()
        } catch Error as e {
            return
        }
        if (whr.Status != 200) {
            ModelLog.Value .= "`n[ERROR] Download failed: " . whr.Status
            SendMessage(0x0115, 7, 0, ModelLog.Hwnd, "A")
            return
        }
        rawResponse := whr.ResponseText
    }
'''
old_download_init = '''    finalUrl := "https://generativelanguage.googleapis.com/v1beta/" . outputUri . ":download?alt=media&key=" . API_KEY

    whr := ComObject("WinHttp.WinHttpRequest.5.1")
    whr.SetTimeouts(30000, 60000, 600000, 600000)
    try {
        whr.Open("GET", finalUrl, false)
        whr.Send()
    } catch Error as e {
        return
    }

    if (whr.Status != 200) {
        ModelLog.Value .= "`n[ERROR] Download failed: " . whr.Status
        SendMessage(0x0115, 7, 0, ModelLog.Hwnd, "A") ; WM_VSCROLL = 0x0115, SB_BOTTOM = 7
        return
    }


    rawResponse := whr.ResponseText'''
content = content.replace(old_download_init, download_curl)

# 7. Update TestAPIConnection for curl
old_tac = '''TestAPIConnection(*) {
    ;Status_Text.Value := "Fetching models..." ;
    ModelLog.Value .= "`nFetching models..."
    SendMessage(0x0115, 7, 0, ModelLog.Hwnd, "A") ; WM_VSCROLL = 0x0115, SB_BOTTOM = 7
    Prog_Bar.Value := 10 ;

    try {
        whr := ComObject("WinHttp.WinHttpRequest.5.1")
        ; Using your GET request logic to verify the key
        url := "https://generativelanguage.googleapis.com/v1beta/models?key=" . API_KEY

        whr.Open("GET", url, false)
        whr.Send()

        if (whr.Status == 200) {
            Prog_Bar.Value := 100 ;
            ;Status_Text.Value := "Models Found" ;

            ; Parse names and format for the Edit Control
            modelList := ""
            pos := 1
            while (pos := RegExMatch(whr.ResponseText, "`"name`":\\s*`"models/([^`"]+)`"", &match, pos + 1)) {
                modelList .= match[1] . "`r`n"
            }'''

new_tac = '''TestAPIConnection(*) {
    global useCurl, API_KEY
    ModelLog.Value .= "`nFetching models..."
    SendMessage(0x0115, 7, 0, ModelLog.Hwnd, "A")
    Prog_Bar.Value := 10

    try {
        url := "https://generativelanguage.googleapis.com/v1beta/models?key=" . API_KEY
        responseText := ""
        status := 0
        if (useCurl) {
            resFile := A_Temp . "\\\\gemini_models_" . A_TickCount . ".json"
            curlCmd := 'curl -s "' . url . '" -o "' . resFile . '"'
            RunWait(curlCmd, , "Hide")
            if FileExist(resFile) {
                responseText := FileRead(resFile)
                FileDelete(resFile)
                status := 200
            }
        } else {
            whr := ComObject("WinHttp.WinHttpRequest.5.1")
            whr.Open("GET", url, false)
            whr.Send()
            status := whr.Status
            responseText := whr.ResponseText
        }

        if (status == 200) {
            Prog_Bar.Value := 100
            modelList := ""
            pos := 1
            while (pos := RegExMatch(responseText, "`"name`":\\\\s*`"models/([^`"]+)`"", &match, pos + 1)) {
                modelList .= match[1] . "`r`n"
            }'''
content = content.replace(old_tac, new_tac)

# 8. ProcessNextTask non-blocking completion
old_pnt_end = '''    if (CurrentBatchIndex >= TotalTasks) {
        SetTimer(ProcessNextTask, 0) ; // Stop the timer
        ToggleUI(true)               ; // Re-enable buttons
        return
    }'''
new_pnt_end = '''    if (CurrentBatchIndex >= TotalTasks) {
        SetTimer(ProcessNextTask, 0) ; // Stop the timer
        if (!useCurl || PendingTasks == 0)
            ToggleUI(true)               ; // Re-enable buttons
        return
    }'''
content = content.replace(old_pnt_end, new_pnt_end)
content = content.replace('ProcessNextTask() {', 'ProcessNextTask() {\n    global useCurl, PendingTasks')

# 9. HotIf fix and placement
content = content.replace('#HotIf WinActive("ahk_id " . MyGui.Hwnd)', '#HotIf WinActive("Gemini 2026 Pro Editor")')
content = content.replace('^r:: reload', '^r:: Reload()')

# 10. Curl Helpers
helpers = '''

CheckCurlProgress(pid, resFile, payFile, batchIdx, nameNoExt) {
    global CurlTimers
    static processed := Map()
    if processed.Has(pid)
        return
    resText := ""
    if FileExist(resFile) {
        try { resText := FileRead(resFile) }
    }
    if RegExMatch(resText, 's)"data":\\\\s*"([^"]+)"', &imgMatch) {
        ProcessClose(pid)
        ProcessCurlResult(pid, resFile, payFile, batchIdx, nameNoExt, imgMatch[1], resText)
        processed[pid] := true
        if CurlTimers.Has(pid) {
            SetTimer(CurlTimers[pid], 0)
            CurlTimers.Delete(pid)
        }
        return
    }
    if !ProcessExist(pid) {
        if !processed.Has(pid) {
             if RegExMatch(resText, 's)"data":\\\\s*"([^"]+)"', &imgMatch) {
                 ProcessCurlResult(pid, resFile, payFile, batchIdx, nameNoExt, imgMatch[1], resText)
             } else {
                 ModelLog.Value .= "`n[curl] Task " . batchIdx . " failed or no image data."
                 LV_Tasks.Modify(batchIdx, "", , , , , "Failed")
                 CleanupCurlTask(pid, resFile, payFile)
             }
             processed[pid] := true
             if CurlTimers.Has(pid) {
                 SetTimer(CurlTimers[pid], 0)
                 CurlTimers.Delete(pid)
             }
        }
    }
}

ProcessCurlResult(pid, resFile, payFile, batchIdx, nameNoExt, b64Data, fullRes) {
    try {
        binData := Base64ToBin(b64Data)
        finalExt := (InStr(fullRes, "image/png")) ? "png" : "jpg"
        outPath := OutputDir . "\\\\" . nameNoExt . "_" . A_Now . "." . finalExt
        SaveBinaryImage(binData, outPath)
        ModelLog.Value .= "`n[curl] Saved: " . outPath
        LV_Tasks.Modify(batchIdx, "", , , , , "Success")
    } catch Error as e {
        ModelLog.Value .= "`n[curl] Error: " . e.Message
        LV_Tasks.Modify(batchIdx, "", , , , , "Failed")
    }
    CleanupCurlTask(pid, resFile, payFile)
}

CleanupCurlTask(pid, resFile, payFile) {
    global PendingTasks
    PendingTasks -= 1
    try FileDelete(resFile)
    try FileDelete(payFile)
    CheckQueueCompletion()
}

CheckQueueCompletion() {
    global PendingTasks, CurrentBatchIndex, LV_Tasks
    if (PendingTasks == 0 && CurrentBatchIndex >= LV_Tasks.GetCount()) {
        ToggleUI(true)
        ModelLog.Value .= "`n[System] All tasks completed."
        SendMessage(0x0115, 7, 0, ModelLog.Hwnd, "A")
    }
}
'''
if 'CheckCurlProgress' not in content:
    content += helpers

# CRLF normalization
content = content.replace('\\r\\n', '\\n').replace('\\n', '\\r\\n')

with open('nanoV3.ahk', 'wb') as f:
    f.write(content.encode('utf-8'))
