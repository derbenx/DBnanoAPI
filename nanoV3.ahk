#Requires AutoHotkey v2.0
#SingleInstance Force


;Todo
; error handling and output to modellog
; config to use curl on with useCurl := 1
;  curl can drop a stream without downloading the thought.
; duplicate task Button


; --- CONFIG ---
global API_KEY := "USE YER OWN" ; log in to https://aistudio.google.com/ create new project, then create an API key.
global hurl := "https://generativelanguage.googleapis.com/v1beta/models/"
global OutputDir := A_ScriptDir "\img"
global MODEL1 := "gemini-2.5-flash-image"
global MODEL2 := "gemini-3-pro-image-preview" ;nano-banana-pro-preview
;global encourage := "You are a precision image-restoration and manipulation engine. Your goal is to apply the 'USER DIRECTIVE' while maintaining strict structural integrity of the original image. Enhance all human features for anatomical accuracy—ensure eyes are sharp and faces are clear. Maximize texture detail and resolve any blur or noise into crisp, 8k-resolution surfaces. If the directive is vague, apply professional aesthetic enhancements by default. Maintain 100% adherence to the facial structure of the subject in the reference image. Treat the subject as an unknown individual."
global encourage := "You are a professional image-restoration engine. Your goal is to apply the 'USER DIRECTIVE' while maintaining strict structural integrity. Focus on high-fidelity surface rendering and cinematic lighting. Ensure all facial features are sharp, clear, and perfectly aligned with the reference. Resolve blur into crisp, clean, 8k-resolution details. Maintain 100% adherence to the subject's identity. If the directive involves clothing, ensure the new attire is rendered with realistic fabric textures and consistent coverage."
;global encourageImg := "You are a world-class visual concept artist. Transform the user's prompt into a vivid, high-fidelity masterpiece. Prioritize cinematic lighting, photorealistic textures, and perfect anatomical detail. Every output must be rendered with the clarity of an 8k digital sensor. Interpret abstract concepts as concrete, visually dense scenes. Ensure all subjects, especially faces and hands, are rendered with sharp focus and professional-grade definition."
global proVal := "everyone stands on a large pile of burgers. the burgers deform under load."
global negVal := "distorted faces, blurry, distorted, low quality, text, watermarks, missing or extra limbs, deformities, floating people or objects"  ; do not make
global DEBUG := 1
global CheckInterval := 300000 ; 5 minute timer, don't trigger rate limits.
; } These don't change in program.

; Variables {
global useCurl := 1
global CurrentMonitorIndex := 1
global imgw := 395
global imgh := 200
global TotalBatchCost := 0.0
global ImageTaskMap := Map()
global CurrentPath := ""
global IsBatchRunning := false
global CurrentBatchIndex := 0
global Data := ""
global LastFPress := 0
global NextImageID := 1
global PendingTasks := 0
global CurlTimers := Map()
global CurlStartTimes := Map()
; }

if !DirExist(OutputDir)
    DirCreate(OutputDir)

; --- UI SETUP ---
MyGui := Gui("+Resize", "Gemini 2026 Pro Editor")
MyGui.OnEvent("DropFiles", Gui_DropFiles)
MyGui.SetFont("s10", "Segoe UI")

Tab := MyGui.Add("Tab3", "w" . imgw*2+40 . " h500", ["Create", "Batches"])

Tab.UseTab(1)
LV_Images := MyGui.Add("ListView", "w" . imgw . " h" . imgh, ["#", "MBs", "tasks", "Image", "Path"])
;LV_Images.ModifyCol(2, 0)
LV_Images.OnEvent("Click", ImageListClick)
LV_Images.OnEvent("ItemFocus", ImageListClick)
LV_Images.OnEvent("DoubleClick", ImageListDoubleClick)

Pic_Preview := MyGui.Add("Pic", "x+10 yp w" . imgw . " h" . imgh . " +Border +Center ", "")

MyGui.SetFont("s10 norm")
LV_Tasks := MyGui.Add("ListView", "x30 y250 w" . imgw*2+10 . " h140", ["Img", "Agent", "Res", "Ratio", "Status", "Cost ($)", "Prompt", "TaskIdx"])
LV_Tasks.ModifyCol(8, 0)
LV_Tasks.OnEvent("Click", TaskListClick)
LV_Tasks.OnEvent("DoubleClick", ShowTaskForm)

MyGui.SetFont("s12 bold")
Btn_Add := MyGui.Add("Button", "w110 h30 Disabled", "Add Task")
Btn_Add.OnEvent("Click", ShowTaskForm)
Btn_Gen := MyGui.Add("Button", "x+10 yp w100 h30", "Generate")
Btn_Gen.OnEvent("Click", AddNullImage)
TotalCostDisplay := MyGui.Add("Text", "x+10 yp w120", "Total: $0.0000")
Radio_Immediate := MyGui.Add("Radio", "x+10 yp Checked", "Immediate")
Radio_Immediate.OnEvent("Click", RefreshAllCosts) ; Add this trigger
Radio_Batch := MyGui.Add("Radio", "x+20", "Batch")
Radio_Batch.OnEvent("Click", RefreshAllCosts) ; Add this trigger

Btn_load := MyGui.Add("Button", "x30 yp+40 w100 h30", "Load CSV")
Btn_load.OnEvent("Click", LoadCSV)
Btn_save := MyGui.Add("Button", "x+10 yp w100 h30", "Save CSV")
Btn_save.OnEvent("Click", SaveCSV)
Btn_Test := MyGui.Add("Button", "x+10 yp w100 h30", "Test API Key")
Btn_Test.OnEvent("Click", TestAPIConnection)
Btn_Run := MyGui.Add("Button", "x+10 yp w175 h30 Disabled", "RUN IMMEDIATE")
Btn_Run.OnEvent("Click", StartBatch)

Prog_Bar := MyGui.Add("Progress", "x20 y480 w" . imgw*2+20 . " h15 cYellow", 0)

Tab.UseTab(2)
;batView := MyGui.Add("ListView", "x20 y50 w" . imgw*2+40 . " h380 Grid", ["File", "Agent", "Res", "Status", "Load", "JobID"])
batView := MyGui.Add("ListView", "x20 y50 w" . imgw*2+40 . " h380 Grid", ["JobID", "Status", "Submitted", "Progress"])
Btn_ClearBatches := MyGui.Add("Button", "x20 yp+390 w150 h30", "Clear Finished")
Btn_ClearBatches.OnEvent("Click", ClearFinishedJobs)
batBar := MyGui.Add("Progress", "x20 y480 w" . imgw*2+20 . " h15 cGreen", 0)

Tab.UseTab()
global ModelLog := MyGui.Add("Edit", "xm y500 w" . imgw*2+40 . " r5 +ReadOnly +vModelLog", "")
MyGui.Show()

ModelLogMsg("Networking initialized. useCurl=" . useCurl . " (1=curl, 0=WinHttp)")
SetTimer(LoadExistingJobs, -500)
if (useCurl && !FileExist(A_ScriptDir . "\curl.exe") && !FileExist(A_WinDir . "\System32\curl.exe")) {
    useCurl := 0
    ModelLog.Value .= "`n[" . FormatTime(, "HH:mm:ss") . "] Cannot find curl.exe using standard mode."
} else {
    ModelLog.Value .= "`n[" . FormatTime(, "HH:mm:ss") . "] Found curl.exe using curl mode."
}

SaveCSV(*) {
    savePath := FileSelect("S16", A_ScriptDir, "Save Task Configuration", "CSV (*.csv)")
    if (savePath != "" && !RegExMatch(savePath, "i)\.csv$"))
        savePath .= ".csv"
    if !RegExMatch(savePath, "i)\.csv$")
        savePath .= ".csv"
    if (savePath == "")
        return

    if !RegExMatch(savePath, "i)\.csv$")
        savePath .= ".csv"

    try {
        fileObj := FileOpen(savePath, "w", "UTF-8")

        Loop LV_Images.GetCount() {
            imgID := LV_Images.GetText(A_Index, 1)
            filePath := LV_Images.GetText(A_Index, 5)

            ; Write the image path line: img, index, path
            fileObj.WriteLine("img," . imgID . "," . filePath)

            if ImageTaskMap.Has(imgID) {
                for task in ImageTaskMap[imgID] {
                    ; Cleaning BOTH prompts of commas to prevent column shifting
                    cleanPrompt := StrReplace(task.Prompt, ",", "¢")
                    cleanNeg    := StrReplace(task.NegativePrompt, ",", "¢")
                    cleanPrompt := StrReplace(StrReplace(cleanPrompt, "`r", " "), "`n", " ") ; no lf cr
                    cleanNeg := StrReplace(StrReplace(cleanNeg, "`r", " "), "`n", " ") ; no lf cr

                    ; New structure: tsk, parentIdx, size, agent, ratio, prompt, negPrompt, format
                    line := "tsk," . imgID . "," . task.Size . "," . task.Agent . "," . task.Ratio . "," . cleanPrompt . "," . cleanNeg . "," . task.Format
                    fileObj.WriteLine(line)
                }
            }
        }
        fileObj.Close()
        ModelLog.Value .= "`n[" . FormatTime(, "HH:mm:ss") . "] Configuration saved with prompt sanitization."
        SendMessage(0x0115, 7, 0, ModelLog.Hwnd, "A") ; WM_VSCROLL = 0x0115, SB_BOTTOM = 7
    } catch Error as e {
        MsgBox "Save failed: " . e.Message
    }
}

LoadCSV(*) {
    loadPath := FileSelect(3, A_ScriptDir, "Open Task Configuration", "CSV (*.csv)")
    if (loadPath == "")
        return

    global ImageTaskMap := Map()
    global NextImageID := 1
    LV_Images.Delete()
    LV_Tasks.Delete()
    tempImgMap := Map() ; Link CSV index to path

    try {
        Loop Read, loadPath {
            parts := StrSplit(A_LoopReadLine, ",")
            if (parts.Length < 3)
               continue

            if (parts[1] == "img") {
                idx := parts[2]
                path := parts[3]
                if (path == "<GENERATE>" || FileExist(path)) {
                    if (path == "<GENERATE>") {
                        fn := "GENERATE"
                        sizeMB := "0.00"
                    } else {
                        SplitPath path, &fn
                        sizeMB := Format("{:.2f}", FileGetSize(path) / 1024 / 1024)
                    }
                    ix := String(NextImageID++)
                    LV_Images.Add(, ix, sizeMB, 0, fn, path)
                    ImageTaskMap[ix] := []
                    tempImgMap[idx] := ix
                }
            }
            else if (parts[1] == "tsk") {
                parentIdx := parts[2]
                if tempImgMap.Has(parentIdx) {
                    currentIx := tempImgMap[parentIdx]

                    newTask := {
                        Size: parts[3],
                        Agent: parts[4],
                        Ratio: parts[5],
                        Prompt: StrReplace(parts[6], "¢", ","),
                        NegativePrompt: StrReplace(parts[7], "¢", ","),
                        Format: parts[8],
                        Status: "Pending",
                        SourcePath: "", ; Will be set below
                        Cost: CalculateCost(parts[4], parts[3]),
                        Mode: Radio_Batch.Value ? "Batch" : "Immediate"
                    }

                    ; Retrieve the SourcePath from the ListView using the current image index
                    Loop LV_Images.GetCount() {
                        if (LV_Images.GetText(A_Index, 1) == currentIx) {
                            newTask.SourcePath := LV_Images.GetText(A_Index, 5)
                            break
                        }
                    }

                    ImageTaskMap[currentIx].Push(newTask)
                    ; Update task count in ListView
                    Loop LV_Images.GetCount() {
                        if (LV_Images.GetText(A_Index, 1) == currentIx) {
                            LV_Images.Modify(A_Index, "", , , ImageTaskMap[currentIx].Length)
                            break
                        }
                    }
                }
            }
        }
        ; UI Refresh logic
        if (LV_Images.GetCount() > 0) {
            LV_Images.Modify(1, "Select Focus")
            ImageListClick(LV_Images, 1)
        }
        UpdateTotalDisplay()
        RefreshTaskTable()
        UpdateButtonStates()
    } catch Error as e {
        MsgBox "Load failed: " . e.Message
    }

; --- 1. THE SCALING FIX ---
}

UpdatePreview(ImgPath) {
    if (ImgPath == "<GENERATE>") {
        Pic_Preview.Value := ""
        return
    }
    if (ImgPath == "" || !FileExist(ImgPath)) {
        return
    }
    temp := Gui()
    pic := temp.Add("Pic",, ImgPath)
    pic.GetPos(,, &iw, &ih)
    temp.Destroy()
    if (iw/ih > imgw/imgh) {
        Pic_Preview.Value := "*w" . imgw*(A_ScreenDPI/96) . " *h-1 " . ImgPath
    } else {
        Pic_Preview.Value := "*w-1 *h" . imgh*(A_ScreenDPI/96) . " " . ImgPath
    }
}

RefreshAllCosts(*) {
    Btn_Run.Text := Radio_Batch.Value ? "RUN BATCH" : "RUN IMMEDIATE"
    ; Loop through every image path in your map
    for path, tasks in ImageTaskMap {
        ; Loop through every task for that image
        for t in tasks {
            ; Re-calculate the cost based on the current toggle state
            t.Cost := CalculateCost(t.Agent, t.Size)
            ; Update the mode of the task to match the new toggle
            t.Mode := Radio_Batch.Value ? "Batch" : "Immediate"
        }
    }

    ; Update the "Total: $0.0000" text
    UpdateTotalDisplay()
    ; Update the ListView rows to show the new prices
    RefreshTaskTable()
}

LoadExistingJobs() {
    jobFile := A_ScriptDir "\jobs.txt"
    if !FileExist(jobFile)
        return

    jobCount:=0
    Loop Read, jobFile {
        if (A_LoopReadLine == "")
         continue

        ;batView.Add(, A_LoopReadLine, "Checking...", A_LoopFileTimeModified, "0%")
        batView.Add(, A_LoopReadLine, "Checking...", "Prior Session", "0%")
        foundList .= "`n  - " . A_LoopReadLine
        jobCount++
    }
    batView.ModifyCol()

    if (jobCount == 0) {
        ModelLog.Value .= "`n[" . FormatTime(, "HH:mm:ss") . "] jobs.txt is empty. No jobs found."
        SendMessage(0x0115, 7, 0, ModelLog.Hwnd, "A")
        return
    }

    ; Success: Log the list for the user to see in the status window
    ModelLog.Value .= "`n[" . FormatTime(, "HH:mm:ss") . "] Found Jobs:" . foundList
    SendMessage(0x0115, 7, 0, ModelLog.Hwnd, "A")

    global NextCheckTime := A_TickCount
    SetTimer(UpdateMonitorProgress, 1000)
}

CalculateCost(agent, res) {
    base := (agent = "Nano Flash") ? 0.039 : (res = "4K") ? 0.24 : 0.134
    ; Apply 50% discount if Batch Mode is selected
    return Radio_Batch.Value ? (base * 0.5) : base
}

; --- 2. DYNAMIC TASK FORM ---

ShowTaskForm(*) {
    global proVal, negVal

    ; // 1. Identify which images are selected
    selectedRows := []
    row := 0
    while (row := LV_Images.GetNext(row)) {
        selectedRows.Push(row)
    }

    if (selectedRows.Length == 0) {
        MsgBox "Please select at least one image on the left first!"
        return
    }

    ; // 2. Determine if we are editing
    isEdit := (Btn_Add.Text == "Edit Task")
    selectedTaskRow := LV_Tasks.GetNext(0, "Focused")

    imgIDDisplay := ""
    currentImgPath := ""

    if (isEdit && selectedTaskRow > 0) {
        imgIDDisplay := LV_Tasks.GetText(selectedTaskRow, 1)
    } else {
        for r in selectedRows {
            id := LV_Images.GetText(r, 1)
            imgIDDisplay .= (imgIDDisplay == "" ? "" : "+") . id
        }
    }

    ; Use the focused image for ratio detection if possible
    focusedRow := LV_Images.GetNext(0, "Focused")
    if (focusedRow > 0)
        currentImgPath := LV_Images.GetText(focusedRow, 5)
}
    else if (selectedRows.Length > 0)
        currentImgPath := LV_Images.GetText(selectedRows[1], 5)

    ; // Detect Aspect Ratio from the actual file
    detectedRatio := "1:1"
    if (currentImgPath != "" && currentImgPath != "<GENERATE>") {
        try {
            temp := Gui()
            pic := temp.Add("Pic",, currentImgPath)
            pic.GetPos(,, &iw, &ih)
            temp.Destroy()
            detectedRatio := GetClosestRatio(iw, ih)
        } catch { }
    }

    tierChoice := 1    ; Default: Nano Flash 1K
    ratioChoice := 5   ; Default: 1:1 (index 5 in the list below)
    formatChoice := 1  ; Default: JPG
    localIdx := 0      ; This will track the task's position inside the Map array

    ; // Set ratio default based on image detection
    ratioList := ["9:16","4:5","3:4","2:3", "1:1", "3:2", "4:3","5:4","16:9","21:9"]
    for i, val in ratioList {
        if (val == detectedRatio)
            ratioChoice := i
    }
    pVal := ""
    nVal := ""

    ; // 3. If Editing, retrieve existing data
    if (isEdit && selectedTaskRow > 0) {
        localIdx := Integer(LV_Tasks.GetText(selectedTaskRow, 8))
        if ImageTaskMap.Has(imgIDDisplay) && ImageTaskMap[imgIDDisplay].Length >= localIdx {
            task := ImageTaskMap[imgIDDisplay][localIdx]
            pVal := task.Prompt
            nVal := task.NegativePrompt

            tierStr := task.Agent . " " . task.Size
            for i, val in ["Nano Flash 1K", "Nano Pro 1K", "Nano Pro 2K", "Nano Pro 4K"] {
                if (val == tierStr)
                    tierChoice := i
            }

            for i, val in ratioList {
                if (val == task.Ratio)
                    ratioChoice := i
            }

            for i, val in ["JPG", "PNG"] {
                if (val == task.Format)
                    formatChoice := i
            }
        }
    } else {
        pVal := proVal
        nVal := negVal
    }

    ; // 4. Create the Task Configuration GUI
    g := Gui(" ", isEdit ? "Edit Task" : "Task Config") ;
    g.Opt("+Owner" . MyGui.Hwnd)
    MyGui.Opt("+Disabled")
    g.SetFont("s9", "Segoe UI")

    g.Add("Text",, "Positive Prompt (What to add/change):")
    ed := g.Add("Edit", "vPrompt w300 r3", pVal)

    g.Add("Text",, "Negative Prompt (What to avoid):")
    neg := g.Add("Edit", "vNeg w300 r2", nVal)

    g.Add("Text", "xm w145", "Tier:")
    g.Add("Text", "x+10 w145", "Aspect Ratio:")

    tier := g.Add("DropDownList", "vTier xm w145 Choose" . tierChoice, ["Nano Flash 1K", "Nano Pro 1K", "Nano Pro 2K", "Nano Pro 4K"])
    ratio := g.Add("DropDownList", "vRatio x+10 w145 Choose" . ratioChoice, ratioList)

    g.Add("Text", "xm w145", "Output Format:")
    fmt := g.Add("DropDownList", "vFormat xm w145 Choose" . formatChoice, ["JPG", "PNG"])

    btn := g.Add("Button", "Default xm w300 h40", isEdit ? "Update Task" : "Confirm Task")

    ; // Button Event
    btn.OnEvent("Click", (*) => (
        SubmitTaskWithExtras(g.Submit(), isEdit, localIdx, imgIDDisplay),
        MyGui.Opt("-Disabled"),
        g.Destroy(),
        MyGui.Show()
    ))

    ; // Handle Window Close (X button)
    g.OnEvent("Close", (*) => (
        MyGui.Opt("-Disabled"),
        g.Destroy(),
        MyGui.Show()
    ))

    g.Show()

ProcessMergedSelection(Prompt, FullTierName, EntryGui) {
    ; Extract Agent and Size from "Nano Pro 4k" -> Agent: "Nano Pro", Size: "4k"
    if RegExMatch(FullTierName, "(.*)\s(\d+K)", &match) {
        Agent := match[1]
        Size := match[2]
        SubmitTaskWithExtras(Data)
    }
}

SubmitTaskWithExtras(Data, isEdit := false, editIndex := 0, taskID := "") {
    ; // Extract Agent and Size from the Tier string (e.g., "Nano Pro 4K")
    RegExMatch(Data.Tier, "(.*)\s(\d+K)", &match)

    ; // 1. Collect paths and IDs for the task
    imgID := taskID
    fullPaths := ""

    if (imgID == "") {
        selectedImageRow := LV_Images.GetNext(0, "Focused")
        if (selectedImageRow > 0) {
            imgID := LV_Images.GetText(selectedImageRow, 1)
            fullPaths := LV_Images.GetText(selectedImageRow, 5)
        }
    } else {
        ids := StrSplit(imgID, "+")
        for id in ids {
            Loop LV_Images.GetCount() {
                if (LV_Images.GetText(A_Index, 1) == id) {
                    path := LV_Images.GetText(A_Index, 5)
                    fullPaths .= (fullPaths == "" ? "" : "|") . path
                    break
                }
            }
        }
    }

    if (imgID == "")
        return

    ; // 2. Create the task object
    newTask := {
        ID: imgID,
        Prompt: Data.Prompt,
        NegativePrompt: Data.Neg,
        Agent: match[1],
        Size: match[2],
        Ratio: Data.Ratio,
        Format: Data.Format,
        Status: "Pending",
        SourcePath: fullPaths,
        Cost: CalculateCost(match[1], match[2])
    }

    ; // 3. Store in the Map using the Task ID (could be "1" or "1+2")
    if (isEdit && editIndex > 0) {
        ImageTaskMap[imgID][editIndex] := newTask
    } else {
        if !ImageTaskMap.Has(imgID)
            ImageTaskMap[imgID] := []
        ImageTaskMap[imgID].Push(newTask)

        ; Update task count in LV_Images for ALL involved images
        ids := StrSplit(imgID, "+")
        for id in ids {
            Loop LV_Images.GetCount() {
                if (LV_Images.GetText(A_Index, 1) == id) {
                    taskCount := 0
                    for mID, tasks in ImageTaskMap {
                        if (IsIDInMergedID(id, mID)) {
                            taskCount += tasks.Length
                        }
                    }
                    LV_Images.Modify(A_Index, "", , , taskCount)
                    break
                }
            }
        }
    }

    UpdateTotalDisplay()
    RefreshTaskTable()
    UpdateButtonStates()
}

UpdateTotalDisplay() {
    global TotalBatchCost := 0.0
    for path, tasks in ImageTaskMap {
        for t in tasks {
            TotalBatchCost += t.Cost
        }
    }
    TotalCostDisplay.Value := "Total: $" . Format("{:.4f}", TotalBatchCost)
}

RefreshTaskTable() {
    LV_Tasks.Delete()

    ; Loop through the Map by ID
    for imgID, tasks in ImageTaskMap {
        for i, t in tasks {
            ; Column 1 clearly shows which Image ID this task belongs to
            ; Col 5 is Status, Col 6 is Cost, Col 7 is Prompt, Col 8 is TaskIdx
            LV_Tasks.Add(, imgID, t.Agent, t.Size, t.Ratio, t.Status, Format("{:.3f}", t.Cost), t.Prompt, i)
        }
    }
    LV_Tasks.ModifyCol()
    LV_Tasks.ModifyCol(8, 0) ; Keep hidden
}
SubmitBatchJob(fileUri) {
    global useCurl, API_KEY
    selectedModel := Radio_Immediate.Value ? MODEL1 : MODEL2
    apiUrl := "https://generativelanguage.googleapis.com/v1beta/models/" . selectedModel . ":batchGenerateContent?key=" . API_KEY

    RegExMatch(fileUri, "files/[^/`"]+", &match)
    fileId := match ? match[0] : fileUri
    payload := '{ "batch": { "input_config": { "file_name": "' . fileId . '" } } }'

    responseText := ""
    status := 0

    if (useCurl) {
        resFile := A_Temp . "\\gemini_batch_sub_" . A_TickCount . ".json"
        payloadFile := A_Temp . "\\gemini_batch_sub_pay_" . A_TickCount . ".json"
        FileAppend(payload, payloadFile, "UTF-8-RAW")

        curlCmd := 'curl -s -X POST "' . apiUrl . '" -H "Content-Type: application/json" -d "@' . payloadFile . '" -o "' . resFile . '"'
        Run(curlCmd, , "Hide", &pid)
        while ProcessExist(pid)
            Sleep(50)

        if FileExist(resFile) {
            responseText := FileRead(resFile)
            status := 200
            FileDelete(resFile)
        }
        if FileExist(payloadFile)
            FileDelete(payloadFile)

        if (InStr(responseText, '"error"'))
            status := 400
    } else {
        whr := ComObject("WinHttp.WinHttpRequest.5.1")
        whr.SetTimeouts(30000, 60000, 600000, 600000)
        whr.Open("POST", apiUrl, false)
        whr.SetRequestHeader("Content-Type", "application/json")
        whr.Send(payload)
        status := whr.Status
        responseText := whr.ResponseText
    }

    if (status != 200 || InStr(responseText, '"error"'))
        throw Error("Batch Submission Failed (" . status . "): " . responseText)

    jobID := JSON_Get(responseText, "name")

    FileAppend(jobID . "`n", A_ScriptDir . "\\jobs.txt")

    if (true) {
        global NextCheckTime := A_TickCount
        SetTimer(UpdateMonitorProgress, 1000)
    }
    return jobID
}

CreateBatchFile(TaskMap) {
    batchPath := A_ScriptDir "\batch_job.jsonl"
    if FileExist(batchPath)
        FileDelete(batchPath)

    fileObj := FileOpen(batchPath, "w", "UTF-8-RAW")
    selectedModel := (Radio_Immediate.Value) ? MODEL1 : MODEL2
    modelPath := "models/" . selectedModel

    for imgID, tasks in TaskMap {
        for task in tasks {
            ; Use the specific SourcePath saved with this task
            currentTaskPath := task.SourcePath
            fn := StrReplace(currentTaskPath, "\", "_")

            ; Pass the specific task's path to the payload creator
            payload := CreateJsonPayload(task, currentTaskPath)
            payload := Trim(payload)

            ; Aggressive flattening for JSONL compliance [cite: 598, 602]
            payload := RegExReplace(payload, "[\r\n\t]+", " ")
            payload := RegExReplace(payload, "\s+", " ")
            payload := Trim(payload)

            line := '{"custom_id": "' . fn . '", "request": {"model": "' . modelPath . '", ' . SubStr(payload, 2) . '}'
            fileObj.WriteLine(line)
        }
    }
    fileObj.Close()
    return batchPath
}

AddNullImage(*) {
    global NextImageID
    ix := String(NextImageID++)
    LV_Images.Add(, ix, "0.00", 0, "GENERATE", "<GENERATE>")
    if !ImageTaskMap.Has(ix) {
        ImageTaskMap[ix] := []
    }
    LV_Images.Modify(LV_Images.GetCount(), "Select Focus")
    ImageListClick(LV_Images, LV_Images.GetCount())
    UpdateButtonStates()
}

Gui_DropFiles(GuiObj, GuiCtrlObj, FileArray, X, Y) {
    global NextImageID
    for i, file in FileArray {
        SplitPath file, &fn
        ix := String(NextImageID++)
        sizeMB := Format("{:.2f}", FileGetSize(file) / 1024 / 1024)
        LV_Images.Add(, ix, sizeMB, 0, fn, file)

        if !ImageTaskMap.Has(ix) {
            ImageTaskMap[ix] := [] ;
        }
    }
    LV_Images.ModifyCol(1, "AutoHdr")
    if (LV_Images.GetCount() > 0) {
        LV_Images.Modify(1, "Select Focus") ; Select the first row
        ImageListClick(LV_Images, 1)        ; Trigger the preview and task list logic
    }
    LV_Images.ModifyCol()
    UpdateButtonStates()
}

TaskListClick(LV, RowNum) {
    if (RowNum > 0) {
        Btn_Add.Text := "Edit Task"
    } else {
        Btn_Add.Text := "Add Task"
    }
}

ImageListClick(LV, RowNum) {
    if (RowNum <= 0 || RowNum > LV.GetCount())
        return ;

    ; Reset button to "Add Task" when switching images
    Btn_Add.Text := "Add Task"

    try {
        fullPath := LV.GetText(RowNum, 5) ; // Column 5 is Full Path
        if (fullPath != "") {
            if (fullPath == "<GENERATE>" || FileExist(fullPath)) {
                global CurrentPath := fullPath
                UpdatePreview(fullPath) ;
                RefreshTaskTable() ;
            }
        }
    } catch {
        return
    }
    UpdateButtonStates()
}

ImageListDoubleClick(LV, RowNum) {
    if (RowNum <= 0 || RowNum > LV.GetCount())
        return

    newPath := FileSelect(3, A_ScriptDir, "Select New Image", "Images (*.jpg; *.png; *.jpeg)")
    if (newPath == "")
        return

    SplitPath newPath, &fn
    sizeMB := Format("{:.2f}", FileGetSize(newPath) / 1024 / 1024)
    imgID := LV.GetText(RowNum, 1)

    ; Update ListView: Col 2 (MBs), Col 4 (Image), Col 5 (Path)
    LV.Modify(RowNum, "", , sizeMB, , fn, newPath)

    ; Update SourcePath in ImageTaskMap for ALL tasks involving this image
    for mID, tasks in ImageTaskMap {
        if (IsIDInMergedID(imgID, mID)) {
             newPaths := ""
             ids := StrSplit(mID, "+")
             for id in ids {
                 path := ""
                 Loop LV_Images.GetCount() {
                     if (LV_Images.GetText(A_Index, 1) == id) {
                         path := LV_Images.GetText(A_Index, 5)
                         break
                     }
                 }
                 newPaths .= (newPaths == "" ? "" : "|") . path
             }
             for t in tasks {
                 t.SourcePath := newPaths
             }



    ; Trigger click logic to update preview and paths
    ImageListClick(LV, RowNum)


ToggleUI(Enable := true) {
    Btn_Run.Enabled := Enable
    Btn_Add.Enabled := Enable
    Radio_Immediate.Enabled := Enable
    Radio_Batch.Enabled := Enable
    ;Status_Text.Value := Enable ? "Ready" : "Processing... Please wait."
}

StartBatch(*) {
    if (LV_Tasks.GetCount() == 0) {
        MsgBox "No tasks to run!"
        return
    }
    if (Radio_Batch.Value) {
        firstAgent := "" ; Changed from firstModel
        isMixed := false

        for filePath, taskList in ImageTaskMap {
            for taskObj in taskList {
                if (firstAgent == "") {
                    firstAgent := taskObj.Agent ; Access .Agent instead of .Model
                } else if (taskObj.Agent != firstAgent) {
                    isMixed := true
                    break 2
                }
            }
        }

        if (isMixed) {
            result := MsgBox("Warning: Your batch contains a mix of models (Flash and Pro).`n`nGoogle Batch API requires all tasks in a single job to use the SAME model.")
            return
        }
    }
    ; --- END OF CHECK ---

    ToggleUI(false)

    if (Radio_Batch.Value) {
        ModelLog.Value .= "`n[" . FormatTime(, "HH:mm:ss") . "] Starting Batch Upload..."
        SendMessage(0x0115, 7, 0, ModelLog.Hwnd, "A") ; WM_VSCROLL = 0x0115, SB_BOTTOM = 7
        SetLoadingState(true)

        try {
            batchPath := CreateBatchFile(ImageTaskMap)

            ; --- LOG FILE UPDATE ---
            ;FileAppend("`n[" . FormatTime(, "HH:mm:ss") . "] BATCH START: File created at " . batchPath . "`n", "debug.log")


            batView.Add(, jobName, "Submitted", FormatTime(, "HH:mm:ss"), "0%")
            batView.ModifyCol()
            global NextCheckTime := A_TickCount ; batch check
            SetTimer(UpdateMonitorProgress, 1000)

            ModelLog.Value .= "`n[" . FormatTime(, "HH:mm:ss") . "] Batch Submitted: " . jobName
            SendMessage(0x0115, 7, 0, ModelLog.Hwnd, "A") ; WM_VSCROLL = 0x0115, SB_BOTTOM = 7
            SetLoadingState(false)
            Prog_Bar.Value := 100
            ToggleUI(true)

        } catch Error as e {
            if (DEBUG)
                ;FileAppend("[" . FormatTime(, "HH:mm:ss") . "] BATCH CRITICAL ERROR: " . e.Message . "`n", "debug.log")


; Function to start/stop the jumping

SetLoadingState(active) {
    if (active) {
        SetTimer(FakeProgress, 200) ; Jump every 200ms
    } else {
        SetTimer(FakeProgress, 0)   ; Stop jumping
        Prog_Bar.Value := 0
    }
}

ProcessNextTask() {
    global useCurl, PendingTasks, CurrentBatchIndex, ImageTaskMap
    TotalTasks := LV_Tasks.GetCount()

    if (CurrentBatchIndex >= TotalTasks) {
        SetTimer(ProcessNextTask, 0)
        if (!useCurl || PendingTasks <= 0) {
            ToggleUI(true)
            global IsBatchRunning := false
        }
        return
    }

    CurrentBatchIndex++
    imgID := LV_Tasks.GetText(CurrentBatchIndex, 1)
    if (imgID == "")
        return

    localIdx := 0
    tempCounter := 0
    found := false
    for mID, tasks in ImageTaskMap {
        for idx, t in tasks {
            tempCounter++
            if (tempCounter == CurrentBatchIndex) {
                localIdx := idx
                imgID := mID
                found := true
                break 2
            }
        }
    }

    if (found) {
        try {
            targetTask := ImageTaskMap[imgID][localIdx]
            RunGeminiTask(targetTask.SourcePath, targetTask, CurrentBatchIndex)
        } catch Error as e {
            ModelLogMsg("Task execution error: " . e.Message)
        }
    }
} else {
         else if InStr(fullPath, "|") {
        nameNoExt := "Merged_" . StrReplace(taskObj.ID, "+", "_")
    } else {
        SplitPath fullPath, &nameWithExt, &dir, &ext, &nameNoExt
    }

    MODEL_ID := InStr(agent, "Flash") ? MODEL1 : MODEL2
    if (useCurl) {
        payload := CreateJsonPayload(taskObj, fullPath)
        payloadFile := A_Temp . "\\gemini_pay_" . A_TickCount . "_" . batchIdx . ".json"
        responseFile := A_Temp . "\\gemini_res_" . A_TickCount . "_" . batchIdx . ".json"
        try { if FileExist(payloadFile) { FileDelete(payloadFile) } } catch { }
        FileAppend(payload, payloadFile, "UTF-8-RAW")
        apiUrl := hurl . MODEL_ID . ":streamGenerateContent?key=" . API_KEY
        curlCmd := 'curl -s -N -X POST "' . apiUrl . '" -H "Content-Type: application/json" -d "@' . payloadFile . '" -o "' . responseFile . '"'
        Run(curlCmd, , "Hide", &pid)
        global PendingTasks += 1
        CurlTimers[pid] := CheckCurlProgress.Bind(pid, responseFile, payloadFile, batchIdx, nameNoExt)
        SetTimer(CurlTimers[pid], 200)
        ModelLogMsg("[curl] Task " . batchIdx . " started (PID: " . pid . ")")
        return
    }

    try {
        payload := CreateJsonPayload(taskObj, fullPath)
        ModelLogMsg("Info: " . MODEL_ID . " " . taskObj.Ratio . " " . taskObj.Size . " " . nameNoExt)

        whr := ComObject("WinHttp.WinHttpRequest.5.1")
        whr.SetTimeouts(30000, 60000, 600000, 600000)
        apiUrl := hurl . MODEL_ID . ":generateContent?key=" . API_KEY

        whr.Open("POST", apiUrl, false)
        whr.SetRequestHeader("Content-Type", "application/json")
        whr.Send(payload)

        if (whr.Status == 200) {
            responseText := whr.ResponseText
            if RegExMatch(responseText, 's)"data":\s*"([^"]+)"', &imgMatch) {
                binData := Base64ToBin(imgMatch[1])
                finalExt := (InStr(responseText, "image/png")) ? "png" : "jpg"
                outPath := OutputDir . "\\" . nameNoExt . "_" . A_Now . "." . finalExt
                SaveBinaryImage(binData, outPath)
                ModelLogMsg("Saved: " . outPath)
                LV_Tasks.Modify(batchIdx, "", , , , , "Success")
            } else {
                ModelLogMsg("[DROPPED]: Task " . batchIdx . " failed or safety blocked.")
                LV_Tasks.Modify(batchIdx, "", , , , , "Failed")
            }
        } else {
            ModelLogMsg("API Error " . whr.Status . ": " . whr.ResponseText)
            LV_Tasks.Modify(batchIdx, "", , , , , "Failed")
        }
    } catch Error as e {
        ModelLogMsg("Error in RunGeminiTask: " . e.Message)
        LV_Tasks.Modify(batchIdx, "", , , , , "Failed")
    }
}}

SaveBinaryImage(binBuffer, path) {
    try {
        if FileExist(path)
            FileDelete(path)

        fileObj := FileOpen(path, "w", "cp0") ; Open for writing in raw mode
        fileObj.RawWrite(binBuffer)            ; Write the raw buffer directly
        fileObj.Close()
    } catch Error as e {
        if (DEBUG)
            FileAppend("`n[" . FormatTime() . "] SAVE ERROR: " . e.Message, "debug.log")
        throw e
    }


; Helper to convert image file to Base64 string
}

FileToBase64(FilePath) {
    if !FileExist(FilePath)
        return ""

    FileObj := FileOpen(FilePath, "r")
    FileObj.RawRead(BinData := Buffer(FileObj.Length))
    FileObj.Close()

    ; Use Windows Crypt32 to encode
    DllCall("crypt32\CryptBinaryToString", "Ptr", BinData, "UInt", BinData.Size, "UInt", 0x40000001, "Ptr", 0, "UInt*", &Size := 0)
    VarSetStrCapacity(&Base64, Size)
    DllCall("crypt32\CryptBinaryToString", "Ptr", BinData, "UInt", BinData.Size, "UInt", 0x40000001, "Str", Base64, "UInt*", &Size)
    return StrReplace(StrReplace(Base64, "`r"), "`n")
}

CreateJsonPayload(taskObj, taskImagePath) {
    global encourage

    ; Merge instructions into the text prompt since Gemini doesn't support them in config
    fullPrompt := "USER DIRECTIVE: " . TaskObj.Prompt
                . ". Aspect Ratio: " . TaskObj.Ratio
                . ". Avoid: " . TaskObj.NegativePrompt

    ; Sanitize prompt for JSON
    cleanPrompt := StrReplace(fullPrompt, '"', '\"')
    cleanPrompt := StrReplace(cleanPrompt, "`r", "")
    cleanPrompt := StrReplace(cleanPrompt, "`n", " ")

    cleanEncourage := StrReplace(encourage, '"', '\"')
    cleanEncourage := StrReplace(cleanEncourage, "`r", "")
    cleanEncourage := StrReplace(cleanEncourage, "`n", " ")

    icfg := '"aspectRatio": "' . TaskObj.Ratio . '"'
    if (TaskObj.Size != "1K")
        icfg .= ', "image_size": "' . TaskObj.Size . '"'

    if (taskImagePath == "<GENERATE>") {
        payload := '{'
            . '"contents": [{"parts": [{"text": "' . cleanPrompt . '"}]}], '
            . '"system_instruction": {"parts": [{"text": "' . cleanEncourage . '"}]}, '
            . '"safetySettings": ['
                . '{"category": "HARM_CATEGORY_HARASSMENT", "threshold": "BLOCK_NONE"}, '
                . '{"category": "HARM_CATEGORY_HATE_SPEECH", "threshold": "BLOCK_NONE"}, '
                . '{"category": "HARM_CATEGORY_SEXUALLY_EXPLICIT", "threshold": "BLOCK_NONE"}, '
                . '{"category": "HARM_CATEGORY_DANGEROUS_CONTENT", "threshold": "BLOCK_NONE"}'
            . '], '
            . '"generationConfig": {'
                . '"candidate_count": 1, '
                . '"response_modalities": ["IMAGE"], '
                . '"imageConfig": {' . icfg . '}'
            . '}'
        . '}'
        return payload
    }

    imageParts := ""
    paths := StrSplit(taskImagePath, "|")
    for path in paths {
        if (path == "")
            continue
        mime := (TaskObj.Format = "PNG") ? "image/png" : "image/jpeg"
        b64 := FileToBase64(path)
        imageParts .= ', {"inline_data": {"mime_type": "' . mime . '", "data": "' . b64 . '"}}'
    }

    payload := '{'
        . '"contents": [{"parts": ['
            . '{"text": "' . cleanPrompt . '"}'
            . imageParts
        . ']}], '
        . '"system_instruction": {"parts": [{"text": "' . cleanEncourage . '"}]}, '
        . '"safetySettings": ['
            . '{"category": "HARM_CATEGORY_HARASSMENT", "threshold": "BLOCK_NONE"}, '
            . '{"category": "HARM_CATEGORY_HATE_SPEECH", "threshold": "BLOCK_NONE"}, '
            . '{"category": "HARM_CATEGORY_SEXUALLY_EXPLICIT", "threshold": "BLOCK_NONE"}, '
            . '{"category": "HARM_CATEGORY_DANGEROUS_CONTENT", "threshold": "BLOCK_NONE"}'
        . '], '
        . '"generationConfig": {'
            . '"candidate_count": 1, '
            . '"response_modalities": ["IMAGE"], '
            . '"imageConfig": {' . icfg . '}'
        . '}'
    . '}'

    return payload
}

UploadBatchFile(FilePath) {
    global useCurl, API_KEY
    if !FileExist(FilePath)
        throw Error("Batch file not found: " . FilePath)

    fileData := FileRead(FilePath, "RAW")
    boundary := "-------AHKBoundary" . A_TickCount

    ; 1. Construct the multipart parts
    metadata := '{"file": {"display_name": "batch_job_' . A_Now . '"}}'

    bodyStart := "--" . boundary . "`r`n"
              . "Content-Type: application/json; charset=UTF-8`r`n`r`n"
              . metadata . "`r`n"
              . "--" . boundary . "`r`n"
              . "Content-Type: application/json`r`n`r`n"

    bodyEnd := "`r`n--" . boundary . "--`r`n"

    ; 2. Create the combined binary package
    size := (StrPut(bodyStart, "UTF-8") - 1) + fileData.Size + (StrPut(bodyEnd, "UTF-8") - 1)
    combinedBody := Buffer(size)

    offset := 0
    offset += StrPut(bodyStart, combinedBody, "UTF-8") - 1
    DllCall("RtlMoveMemory", "Ptr", combinedBody.Ptr + offset, "Ptr", fileData.Ptr, "Ptr", fileData.Size)

    offset += fileData.Size
    StrPut(bodyEnd, combinedBody.Ptr + offset, "UTF-8")

    if (useCurl) {
        tempBodyFile := A_Temp . "\gemini_upload_" . A_TickCount . ".bin"
        resFile := A_Temp . "\gemini_upload_res_" . A_TickCount . ".json"
        FileOpen(tempBodyFile, "w", "cp0").RawWrite(combinedBody)
        url := "https://generativelanguage.googleapis.com/upload/v1beta/files?key=" . API_KEY
        curlCmd := 'curl -s -X POST "' . url . '" -H "X-Goog-Upload-Protocol: multipart" -H "Content-Type: multipart/related; boundary=' . boundary . '" --data-binary "@' . tempBodyFile . '" -o "' . resFile . '"'
        Run(curlCmd, , "Hide", &pid)
        while ProcessExist(pid)
            Sleep(50)
        resText := FileRead(resFile)
        try {FileDelete(tempBodyFile)}}
    catch { }
        try {FileDelete(resFile)}}
    catch {
        if RegExMatch(resText, '"uri":\s*"([^"]+)"', &match)
            return match[1]
        throw Error("Curl Upload Failed: " . resText)


    ; 3. THE FIX: Convert Buffer to a Safe COM Stream
    ; This prevents the "No such interface" error by providing a standard IStream interface
    pStream := DllCall("shlwapi\SHCreateMemStream", "Ptr", combinedBody.Ptr, "UInt", combinedBody.Size, "Ptr")
    IStream := ComValue(13, pStream) ; 13 = VT_UNKNOWN (IUnknown/IStream)

    whr := ComObject("WinHttp.WinHttpRequest.5.1")
    whr.SetTimeouts(30000, 60000, 600000, 600000)
    url := "https://generativelanguage.googleapis.com/upload/v1beta/files?key=" . API_KEY

    whr.Open("POST", url, false)
    whr.SetRequestHeader("X-Goog-Upload-Protocol", "multipart")
    whr.SetRequestHeader("Content-Type", "multipart/related; boundary=" . boundary)

    ; 4. Send the Stream instead of the Buffer
    whr.Send(IStream)

    if (whr.Status != 200)
        throw Error("Multipart upload failed: " . whr.ResponseText)

    if RegExMatch(whr.ResponseText, '"uri":\s*"([^"]+)"', &match)
        return match[1]

    throw Error("Could not find URI in response: " . whr.ResponseText)
UpdateMonitorProgress() {
    global CurrentMonitorIndex, NextCheckTime, CheckInterval

    if (A_TickCount < NextCheckTime) {
        remaining := NextCheckTime - A_TickCount
        pct := (1 - (remaining / CheckInterval)) * 100
        batBar.Value := pct
        return
    }

    jobList := []
    Loop batView.GetCount() {
        status := Trim(batView.GetText(A_Index, 2))
        if (status ~= "i)^(Submitted|Checking\.\.\.|Processing\.\.\.|ACTIVE|RUNNING|UNKNOWN)$") {
            jobList.Push({row: A_Index, id: batView.GetText(A_Index, 1)})
        }
    }

    if (jobList.Length == 0) {
        SetTimer(UpdateMonitorProgress, 0)
        batBar.Value := 0
        ModelLogMsg("Batch monitor: No active jobs found. Polling stopped.")
        return
    }

    if (CurrentMonitorIndex > jobList.Length) {
        CurrentMonitorIndex := 1
        NextCheckTime := A_TickCount + CheckInterval
        batBar.Value := 0
        ModelLogMsg("Batch monitor: Round complete. Next check in " . CheckInterval//1000 . "s")
        return
    }

    target := jobList[CurrentMonitorIndex]
    Async
}

Async catch Error as e {
        ModelLogMsg("[Error] WinHttp status check failed: " . e.Message)
    }

ProcessBatchStatus(pid, resFile, jobID, targetRow) {
    if !ProcessExist(pid) {
        if CurlTimers.Has(pid) {
            SetTimer(CurlTimers[pid], 0)
            CurlTimers.Delete(pid)
        }

        if FileExist(resFile) {
            responseText := FileRead(resFile)
            FileDelete(resFile)
            HandleBatchStatus(responseText, jobID, targetRow)
        }
    }
}

HandleBatchStatus(responseText, jobID, targetRow) {
    state := JSON_Get(responseText, "state")
    if (state == "") {
        state := "UNKNOWN"


        if (outputUri != "") {
            ModelLogMsg("Job " . jobID . " SUCCEEDED. Starting download...")
            AsyncDownloadBatch(outputUri, targetRow)
        } else {
            ModelLogMsg("[Warning] Job " . jobID . " succeeded but no responsesFile found.")
        }
    }
}

AsyncDownloadBatch(outputUri, targetRow) {
    global useCurl, API_KEY, CurlTimers
    finalUrl := "https://generativelanguage.googleapis.com/v1beta/" . outputUri . ":download?alt=media&key=" . API_KEY

    if (useCurl) {
        resFile := A_Temp . "\gemini_batch_res_" . A_TickCount . "_" . targetRow . ".jsonl"
        curlCmd := 'curl -s -L "' . finalUrl . '" -o "' . resFile . '"'
         catch Error as e {
        ModelLogMsg("[Error] WinHttp download failed: " . e.Message)
    }
}

ProcessBatchDownload(pid, resFile, targetRow) {
    if !ProcessExist(pid) {
        if CurlTimers.Has(pid) {
            SetTimer(CurlTimers[pid], 0)
            CurlTimers.Delete(pid)
        }

        if FileExist(resFile) {
            responseText := FileRead(resFile)
            FileDelete(resFile)
            HandleBatchDownload(responseText, targetRow)
        }
    }
}

HandleBatchDownload(rawResponse, targetRow) {
    global OutputDir
    if (rawResponse == "")
        return

    batView.Modify(targetRow, "", , "Success", , "100%")

    count := 0
    Loop Parse, rawResponse, "`n", "`r" {
        line := Trim(A_LoopField)
        if (line == "")
            continue

        fn := ""
        if RegExMatch(line, '"custom_id":\s*"([^"]+)"', &m)
            fn := m[1]

        b64 := ""
        if RegExMatch(line, '"data":\s*"([^"]+)"', &m)
            b64 := m[1]
}
        else if RegExMatch(line, '"processed_image_data":\s*"([^"]+)"', &m)
            b64 := m[1]

        if (fn != "" && b64 != "") {
            SplitPath(fn, &justFileName)

            ext := "jpg"
            if RegExMatch(line, '"mimeType":\s*"([^"]+)"', &me)
                ext := (InStr(me[1], "png")) ? "png" : "jpg"

            binData := Base64ToBin(b64)
            outPath := OutputDir . "\Batch_" . A_Now . "_" . count+1 . "_" . justFileName
            if !RegExMatch(outPath, "i)\.(jpg|png)$")
                outPath .= "." . ext

            SaveBinaryImage(binData, outPath)
            count++
        }

    ModelLogMsg("Batch download complete. Saved " . count . " images.")




; Helper to convert the API's text response back to an image file

Base64ToBin(Base64Str) {
    ; Calculate the required buffer size
    DllCall("crypt32\CryptStringToBinary", "Str", Base64Str, "UInt", 0, "UInt", 0x1, "Ptr", 0, "UInt*", &Size := 0, "Ptr", 0, "Ptr", 0)
    BinData := Buffer(Size)
    ; Convert Base64 string to raw binary data
    DllCall("crypt32\CryptStringToBinary", "Str", Base64Str, "UInt", 0, "UInt", 0x1, "Ptr", BinData, "UInt*", &Size, "Ptr", 0, "Ptr", 0)
    return BinData
}

TestAPIConnection(*) {
    global useCurl, API_KEY
    ModelLog.Value .= "`nFetching models..."
    SendMessage(0x0115, 7, 0, ModelLog.Hwnd, "A")
    Prog_Bar.Value := 10

    try {
        url := "https://generativelanguage.googleapis.com/v1beta/models?key=" . API_KEY
        responseText := ""
        status := 0
        if (useCurl) {
            resFile := A_Temp . "\gemini_models_" . A_TickCount . ".json"
            curlCmd := 'curl -s "' . url . '" -o "' . resFile . '"'
            Run(curlCmd, , "Hide", &pid)
        while ProcessExist(pid)
            Sleep(50)
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
            while (pos := RegExMatch(responseText, "`"name`":\s*`"models/([^`"]+)`"", &match, pos + 1)) {
                modelList .= match[1] . "`r`n"
            }

            ModelLog.Value .= "`n" . modelList
            SendMessage(0x0115, 7, 0, ModelLog.Hwnd, "A") ; WM_VSCROLL = 0x0115, SB_BOTTOM = 7

            timestamp := FormatTime(, "HH:mm:ss")
            ;FileAppend("`n[" . timestamp . "] --- SUPPORTED MODELS ---`n" . modelList . "`n", "debug.log")

    }
    Btn_Run.Enabled := HasAnyTasks ;
}

GetClosestRatio(w, h) {
    target := w / h
    ratios := ["9:16", "4:5", "3:4", "2:3", "1:1", "3:2", "4:3", "5:4", "16:9", "21:9"]
    bestMatch := "1:1"
    minDiff := 999.0

    for str in ratios {
        parts := StrSplit(str, ":")
        ratioVal := parts[1] / parts[2]
        diff := Abs(target - ratioVal)

        if (diff < minDiff) {
            minDiff := diff
            bestMatch := str
        }
    }
    return bestMatch
}

ClearFinishedJobs(*) {
    ; Start from the bottom and go to 1 to prevent index shifting
    idx := batView.GetCount()
    while (idx > 0) {
        status := batView.GetText(idx, 2) ; Column 2 is "Status"

        ; Check for every possible "finished" string state
        if (status == "Success" || status == "Failed" ) {
            batView.Delete(idx)
        }
        idx-- ; Manually move to the next item up
    }

    ; Sync the jobs.txt file so deleted items don't return on reload [cite: 52, 57]
     catch {
        ; Silent fail if log file is locked
    }

#HotIf WinActive("Gemini 2026 Pro Editor")
^r:: Reload()
$Del:: {
    FocusedCtrl := MyGui.FocusedCtrl

    if (FocusedCtrl == LV_Images) {
        Row := LV_Images.GetNext(0, "Focused")
        if (Row) {
            imgID := LV_Images.GetText(Row, 1)

            ; Identify all task keys that involve this image ID
            involvedKeys := []
            for mID, tasks in ImageTaskMap {
                if (IsIDInMergedID(imgID, mID)) {
                    involvedKeys.Push(mID)
                }
            }

            ; For each involved key, we might need to update other images' task counts
            otherImagesToUpdate := Map()
            for k in involvedKeys {
                parts := StrSplit(k, "+")
                for p in parts {
                    if (p != imgID)
                        otherImagesToUpdate[p] := 1
                }
                ImageTaskMap.Delete(k)
            }

            LV_Images.Delete(Row)

            ; Update task counts for other images that were part of deleted merged tasks
            for oid, val in otherImagesToUpdate {
                Loop LV_Images.GetCount() {
                    if (LV_Images.GetText(A_Index, 1) == oid) {
                        taskCount := 0
                        for mID, tasks in ImageTaskMap {
                            if (IsIDInMergedID(oid, mID)) {
                                taskCount += tasks.Length
                            }
                        }
                        LV_Images.Modify(A_Index, "", , , taskCount)
                        break
                    }
                }
            }

            RefreshTaskTable()

            if (LV_Images.GetCount() > 0) {
                NewRow := (Row > LV_Images.GetCount()) ? LV_Images.GetCount() : Row
                LV_Images.Modify(NewRow, "Select Focus")
                ; ItemFocus event will handle ImageListClick
            }
        }
    }
    else if (FocusedCtrl == LV_Tasks) {
        Row := LV_Tasks.GetNext(0, "Focused")
        if (Row) {
            targetImgID := LV_Tasks.GetText(Row, 1)
            taskIdx := Integer(LV_Tasks.GetText(Row, 8))

            if ImageTaskMap.Has(targetImgID) {
                if (taskIdx > 0 && taskIdx <= ImageTaskMap[targetImgID].Length) {
                    ImageTaskMap[targetImgID].RemoveAt(taskIdx)

                    if (ImageTaskMap[targetImgID].Length == 0)
                        ImageTaskMap.Delete(targetImgID)

                    ; Update task count in LV_Images for ALL involved images
                    ids := StrSplit(targetImgID, "+")
                    for id in ids {
                        Loop LV_Images.GetCount() {
                            if (LV_Images.GetText(A_Index, 1) == id) {
                                taskCount := 0
                                for mID, tasks in ImageTaskMap {
                                    if (IsIDInMergedID(id, mID)) {
                                        taskCount += tasks.Length
                                    }
                                }
                                LV_Images.Modify(A_Index, "", , , taskCount)
                                break
                            }
                        }
                    }
                }
            }

            RefreshTaskTable()
            UpdateTotalDisplay()
            UpdateButtonStates()
        }
    }
}

^f:: {
    global LastFPress
    if (A_TickCount - LastFPress < 500)
        return
    LastFPress := A_TickCount

    global NextCheckTime := A_TickCount ; Reset timer to now
    UpdateMonitorProgress()             ; Manually trigger the check
}
#HotIf
}

IsIDInMergedID(id, mID) {
    sid := String(id)
    smID := String(mID)
    if (smID == sid)
        return true
    for _, p in StrSplit(smID, "+") {
        if (p == sid)
            return true
    }
    return false
}

ModelLogMsg(txt) {
    global ModelLog
    try {
        timestamp := FormatTime(, "HH:mm:ss")
        ModelLog.Value .= "`n[" . timestamp . "] " . txt
        SendMessage(0x0115, 7, 0, ModelLog.Hwnd, "A")
    } catch { }
}

; --- Async Curl Helpers ---

CheckCurlProgress(pid, responseFile, payloadFile, batchIdx, nameNoExt) {
    if !CurlStartTimes.Has(pid)
        CurlStartTimes[pid] := A_TickCount

    if !ProcessExist(pid) {
        CurlStartTimes.Delete(pid)
        ProcessCurlResult(pid, responseFile, payloadFile, batchIdx, nameNoExt)
        return
    }

    if (A_TickCount - CurlStartTimes[pid] > 180000) {
        ModelLogMsg("Curl task " . batchIdx . " timed out. Terminating.")
        ProcessClose(pid)
        CurlStartTimes.Delete(pid)
        ProcessCurlResult(pid, responseFile, payloadFile, batchIdx, nameNoExt)
        return
    }

    if FileExist(responseFile) {
        try {
            fileContent := FileRead(responseFile)
            if (p1 := InStr(fileContent, '"data":')) {
                if (p2 := InStr(fileContent, '"', , p1 + 7)) {
                    if (p3 := InStr(fileContent, '"', , p2 + 1)) {
                        ProcessClose(pid)
                        CurlStartTimes.Delete(pid)
                        ProcessCurlResult(pid, responseFile, payloadFile, batchIdx, nameNoExt)
                        return
                    }
                }
            }
        } catch { }
    }


ProcessCurlResult(pid, responseFile, payloadFile, batchIdx, nameNoExt) {
    try {
        if CurlTimers.Has(pid) {
            SetTimer(CurlTimers[pid], 0)
            CurlTimers.Delete(pid)
        }

        responseText := ""
        if FileExist(responseFile) {
            responseText := FileRead(responseFile)
            FileDelete(responseFile)
        }

        if FileExist(payloadFile)
            FileDelete(payloadFile)

        global PendingTasks -= 1

        if (responseText != "") {
            ; Use InStr/SubStr for robust extraction from potentially huge JSON strings
            p1 := InStr(responseText, '"data":')
            if (p1) {
                p2 := InStr(responseText, '"', , p1 + 7)
                p3 := InStr(responseText, '"', , p2 + 1)
                if (p2 && p3) {
                    base64Data := SubStr(responseText, p2 + 1, p3 - p2 - 1)

                    ; Detect extension
                    mime := "image/png"
                    if RegExMatch(responseText, '"mimeType":\s*"([^"]+)"', &mimeMatch)
                        mime := mimeMatch[1]
                    ext := (InStr(mime, "jpeg") || InStr(mime, "jpg")) ? "jpg" : "png"

                    outPath := OutputDir . "\" . nameNoExt . "_" . A_Now . "." . ext

                    try {
                        size := 0
                        if DllCall("crypt32\CryptStringToBinary", "Str", base64Data, "UInt", 0, "UInt", 1, "Ptr", 0, "UInt*", &size, "Ptr", 0, "Ptr", 0) {
                            buf := Buffer(size)
                            if DllCall("crypt32\CryptStringToBinary", "Str", base64Data, "UInt", 0, "UInt", 1, "Ptr", buf, "UInt*", &size, "Ptr", 0, "Ptr", 0) {
                                FileOpen(outPath, "w").RawWrite(buf)
                                ModelLogMsg("Image saved: " . outPath)
                                LV_Tasks.Modify(batchIdx, "", , , , , "Success")
                            }
                        }
                    } catch as e {
                        ModelLogMsg("Error decoding image: " . e.Message)
                        LV_Tasks.Modify(batchIdx, "", , , , , "Failed")
                    }
                } else {
                    ModelLogMsg("Could not find complete image data in curl response.")
                    LV_Tasks.Modify(batchIdx, "", , , , , "Failed")
                }
            } else {
                if InStr(responseText, "finishReason") {
                    ModelLogMsg("Curl task " . batchIdx . " was blocked or failed.")
                } else {
                    ModelLogMsg("Curl response (no image data).")
                }
                LV_Tasks.Modify(batchIdx, "", , , , , "Failed")
            }
        } else {
            ModelLogMsg("Curl task " . batchIdx . " finished with no output.")
            LV_Tasks.Modify(batchIdx, "", , , , , "Failed")
        }
    } catch as e {
        ModelLogMsg("Critical error in ProcessCurlResult: " . e.Message)
    }

    CheckQueueCompletion()
}}}}

CheckQueueCompletion() {
    global PendingTasks
    if (PendingTasks <= 0) {
        PendingTasks := 0
        ToggleUI(true)
        ModelLogMsg("All tasks completed.")
    }
}


    }

    try {
        if FileExist(jobFile)
            FileDelete(jobFile)

        if (outString != "")
            FileAppend(outString, jobFile)

        ModelLogMsg("jobs.txt updated (cleaned completed jobs).")
    } catch Error as e {
        ModelLogMsg("[Error] Failed to update jobs.txt: " . e.Message)
    }
}









}}}}}
Async else {
        SetTimer(() => Sync catch Error as e {
        ModelLogMsg("[Error] WinHttp status check failed: " . e.Message)
    }
}

ProcessBatchStatus(pid, resFile, jobID, targetRow) {
    if !ProcessExist(pid) {
        if CurlTimers.Has(pid) {
            SetTimer(CurlTimers[pid], 0)
            CurlTimers.Delete(pid)
        }

        if FileExist(resFile) {
            responseText := FileRead(resFile)
            FileDelete(resFile)
            HandleBatchStatus(responseText, jobID, targetRow)
        }
    }
}

HandleBatchStatus(responseText, jobID, targetRow) {
    state := JSON_Get(responseText, "state")
    if (state == "") {
        state := "UNKNOWN"
    }

    batView.Modify(targetRow, "", , state)

    if (state == "SUCCEEDED" || state == "BATCH_STATE_SUCCEEDED") {
        outputUri := JSON_Get(responseText, "responsesFile")
        if (outputUri == "") {
             if RegExMatch(responseText, '"responsesFile":\\s*"([^"]+)"', &m)
                 outputUri := m[1]
        }

        if (outputUri != "") {
            ModelLogMsg("Job " . jobID . " SUCCEEDED. Starting download...")
            AsyncDownloadBatch(outputUri, targetRow)
        }
    }
}

AsyncDownloadBatch(outputUri, targetRow) {
    global useCurl, API_KEY, CurlTimers
    finalUrl := "https://generativelanguage.googleapis.com/v1beta/" . outputUri . ":download?alt=media&key=" . API_KEY

    if (useCurl) {
        resFile := A_Temp . "\\gemini_batch_res_" . A_TickCount . "_" . targetRow . ".jsonl"
        curlCmd := 'curl -s -L "' . finalUrl . '" -o "' . resFile . '"'
        Run(curlCmd, , "Hide", &pid)

        cb := ProcessBatchDownload.Bind(pid, resFile, targetRow)
        CurlTimers[pid] := cb
        SetTimer(cb, 200)
    } else {
        SetTimer(() => SyncDownloadBatch(finalUrl, targetRow), -10)
    }
}

SyncDownloadBatch(finalUrl, targetRow) {
    whr := ComObject("WinHttp.WinHttpRequest.5.1")
    whr.SetTimeouts(30000, 60000, 600000, 600000)
    try {
        whr.Open("GET", finalUrl, false)
        whr.Send()
        if (whr.Status == 200)
            HandleBatchDownload(whr.ResponseText, targetRow)
    } catch Error as e {
        ModelLogMsg("[Error] WinHttp download failed: " . e.Message)
    }
}

ProcessBatchDownload(pid, resFile, targetRow) {
    if !ProcessExist(pid) {
        if CurlTimers.Has(pid) {
            SetTimer(CurlTimers[pid], 0)
            CurlTimers.Delete(pid)
        }

        if FileExist(resFile) {
            responseText := FileRead(resFile)
            FileDelete(resFile)
            HandleBatchDownload(responseText, targetRow)
        }
    }
}

HandleBatchDownload(rawResponse, targetRow) {
    global OutputDir
    if (rawResponse == "")
        return

    batView.Modify(targetRow, "", , "Success", , "100%")

    count := 0
    Loop Parse, rawResponse, "`n", "`r" {
        line := Trim(A_LoopField)
        if (line == "")
            continue

        fn := ""
        if RegExMatch(line, '"custom_id":\\s*"([^"]+)"', &m)
            fn := m[1]

        b64 := ""
        if RegExMatch(line, '"data":\\s*"([^"]+)"', &m)
            b64 := m[1]
        else if RegExMatch(line, '"processed_image_data":\\s*"([^"]+)"', &m)
            b64 := m[1]

        if (fn != "" && b64 != "") {
            SplitPath(fn, &justFileName)
            ext := "jpg"
            if RegExMatch(line, '"mimeType":\\s*"([^"]+)"', &me)
                ext := (InStr(me[1], "png")) ? "png" : "jpg"

            binData := Base64ToBin(b64)
            outPath := OutputDir . "\\Batch_" . A_Now . "_" . count+1 . "_" . justFileName
            if !RegExMatch(outPath, "i)\\.(jpg|png)$")
                outPath .= "." . ext

            SaveBinaryImage(binData, outPath)
            count++
        }
    }
    ModelLogMsg("Batch download complete. Saved " . count . " images.")


    if (A_TickCount - CurlStartTimes[pid] > 180000) {
        ModelLogMsg("Curl task " . batchIdx . " timed out. Terminating.")
        ProcessClose(pid)
        CurlStartTimes.Delete(pid)
        ProcessCurlResult(pid, responseFile, payloadFile, batchIdx, nameNoExt)
        return
    }

    if FileExist(responseFile) {
        try {
            fileContent := FileRead(responseFile)
            if (p1 := InStr(fileContent, '"data":')) {
                if (p2 := InStr(fileContent, '"', , p1 + 7)) {
                    if (p3 := InStr(fileContent, '"', , p2 + 1)) {
                        ProcessClose(pid)
                        CurlStartTimes.Delete(pid)
                        ProcessCurlResult(pid, responseFile, payloadFile, batchIdx, nameNoExt)
                        return
                    }
                }
            }
        } catch { }
    }
}

ProcessCurlResult(pid, responseFile, payloadFile, batchIdx, nameNoExt) {
    try {
        if CurlTimers.Has(pid) {
            SetTimer(CurlTimers[pid], 0)
            CurlTimers.Delete(pid)
        }

        responseText := ""
        if FileExist(responseFile) {
            responseText := FileRead(responseFile)
            FileDelete(responseFile)
        }

        if FileExist(payloadFile)
            FileDelete(payloadFile)

        global PendingTasks -= 1

        if (responseText != "") {
            p1 := InStr(responseText, '"data":')
            if (p1) {
                p2 := InStr(responseText, '"', , p1 + 7)
                p3 := InStr(responseText, '"', , p2 + 1)
                if (p2 && p3) {
                    base64Data := SubStr(responseText, p2 + 1, p3 - p2 - 1)
                    mime := "image/png"
                    if RegExMatch(responseText, '"mimeType":\\s*"([^"]+)"', &mimeMatch)
                        mime := mimeMatch[1]
                    ext := (InStr(mime, "jpeg") || InStr(mime, "jpg")) ? "jpg" : "png"
                    outPath := OutputDir . "\\" . nameNoExt . "_" . A_Now . "." . ext
                    try {
                        binData := Base64ToBin(base64Data)
                        SaveBinaryImage(binData, outPath)
                        ModelLogMsg("Image saved: " . outPath)
                        LV_Tasks.Modify(batchIdx, "", , , , , "Success")
                    } catch as e {
                        ModelLogMsg("Error saving image: " . e.Message)
                        LV_Tasks.Modify(batchIdx, "", , , , , "Failed")
                    }
                }
            } else {
                ModelLogMsg("Curl task " . batchIdx . " finished with no image data.")
                LV_Tasks.Modify(batchIdx, "", , , , , "Failed")
            }
        } else {
            ModelLogMsg("Curl task " . batchIdx . " finished with no output.")
            LV_Tasks.Modify(batchIdx, "", , , , , "Failed")
        }
    } catch as e {
        ModelLogMsg("Critical error in ProcessCurlResult: " . e.Message)
    }
    CheckQueueCompletion()
}

CheckQueueCompletion() {
    global PendingTasks
    if (PendingTasks <= 0) {
        PendingTasks := 0
        ToggleUI(true)
        ModelLogMsg("All tasks completed.")
    }
}

 catch { }
