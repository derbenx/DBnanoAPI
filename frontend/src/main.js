import './style.css';
import {
    GetImages,
    GetTasks,
    SelectAndAddMultipleImages,
    CreateNewImage,
    DeleteImage,
    AddTask,
    DeleteTask,
    RunTasks,
    RunBatch,
    SaveSessionUI,
    LoadSessionUI,
    GetConfig,
    SaveConfig,
    TestConnection,
    AddImages,
    GetImageBase64,
    ChangeImageUI,
    DuplicateTask,
    ToggleTaskDisabled,
    UpdateTask,
    GetCost,
    GetLastGeneratedImage,
    HasGeneratedImage,
    ClearFinishedJobs,
    GetBatchJobs,
    OpenImageFolder
} from '../wailsjs/go/main/App';
import { EventsOn, OnFileDrop } from '../wailsjs/runtime/runtime';

let state = {
    images: [],
    tasks: [],
    config: {},
    activeTab: 'create',
    selectedImageID: null,
    selectedTaskID: null,
    isHoveringImage: false,
    isRunning: false
};

// --- Initialization ---

window.addEventListener('DOMContentLoaded', async () => {
    try {
        state.config = await GetConfig();
        populateSettings(state.config);

        OnFileDrop((x, y, paths) => {
            if (paths && paths.length > 0) {
                console.log("Files dropped:", paths);
                AddImages(paths);
            }
        }, false);

        setupEventListeners();
        await refreshData();
        renderAll();
        addLog("Application Initialized");
    } catch (err) {
        console.error("Initialization error:", err);
    }
});

async function refreshData() {
    state.images = await GetImages() || [];
    state.tasks = await GetTasks() || [];
    state.batchJobs = await GetBatchJobs() || [];

    if (state.selectedImageID && !state.images.find(i => i.ID == state.selectedImageID)) {
        state.selectedImageID = null;
    }
    if (state.selectedTaskID && !state.tasks.find(t => t.ID == state.selectedTaskID)) {
        state.selectedTaskID = null;
    }

    updateRunButtons();
}

function setupEventListeners() {
    EventsOn("log", (msg) => addLog(msg));
    EventsOn("run_started", () => {
        state.isRunning = true;
        updateRunButtons();
    });
    EventsOn("run_finished", () => {
        state.isRunning = false;
        updateRunButtons();
    });
    EventsOn("images_updated", async () => {
        await refreshData();
        renderImageList();
    });
    EventsOn("tasks_updated", async () => {
        await refreshData();
        renderTaskList();
        if (state.selectedTaskID) {
            const task = state.tasks.find(t => t.ID === state.selectedTaskID);
            if (task) populateEditor(task);
        }
    });

    EventsOn("batch_updated", async () => {
        await refreshData();
        renderBatchList();
    });

    EventsOn("batch_timer", (seconds) => {
        const timerCont = document.getElementById('batch-timer-container');
        if (timerCont) timerCont.innerText = `Next check in: ${seconds}s`;
    });

    EventsOn("test_api_started", () => {
        const status = document.getElementById('test-api-status');
        if (status) {
            status.innerText = "Testing...";
            status.style.color = "#3498db";
        }
    });

    EventsOn("test_api_finished", (success, msg) => {
        const status = document.getElementById('test-api-status');
        if (status) {
            status.innerText = success ? "Success" : "Failed";
            status.style.color = success ? "#2ecc71" : "#e74c3c";
        }
    });

    document.querySelectorAll('.tab').forEach(tab => {
        tab.addEventListener('click', () => {
            document.querySelectorAll('.tab').forEach(t => t.classList.remove('active'));
            tab.classList.add('active');
            state.activeTab = tab.dataset.tab;

            document.querySelectorAll('.tab-content').forEach(c => c.style.display = 'none');
            const target = document.getElementById('tab-' + state.activeTab);
            target.style.display = (state.activeTab === 'create' ? 'flex' : 'block');

            const logs = document.getElementById('logs');
            logs.style.display = (state.activeTab === 'settings' ? 'none' : 'block');

            renderAll();
        });
    });

    // Control Buttons
    document.getElementById('btn-load-session').onclick = () => LoadSessionUI();
    document.getElementById('btn-save-session').onclick = () => SaveSessionUI();
    document.getElementById('btn-run-immediate').onclick = () => {
        RunTasks();
        addLog("Running tasks in Immediate mode...");
    };
    document.getElementById('btn-run-batch').onclick = () => {
        RunBatch();
        addLog("Batch mode submission triggered.");
    };

    // Global Exposure
    window.SelectAndAddMultipleImages = SelectAndAddMultipleImages;
    window.CreateNewImage = CreateNewImage;
    window.TestConnection = TestConnection;
    window.DeleteImage = DeleteImage;
    window.DeleteTask = DeleteTask;
    window.ChangeImageUI = ChangeImageUI;
    window.DuplicateTask = DuplicateTask;
    window.ToggleTaskDisabled = ToggleTaskDisabled;
    window.OpenImageFolder = OpenImageFolder;
    window.ShowGeneratedImage = async (id) => {
        const b64 = await GetLastGeneratedImage(id);
        if (b64) {
            document.getElementById('editor-container').style.display = 'none';
            document.getElementById('preview-container').style.display = 'flex';
            const preview = document.getElementById('image-preview');
            preview.innerHTML = `<img src="data:image/jpeg;base64,${b64}" class="preview-image">`;
            state.isHoveringImage = true;
        } else {
            addLog("No generated image found for task " + id);
        }
    };
    window.ClearFinishedJobs = async () => {
        await ClearFinishedJobs();
        await refreshData();
        renderBatchList();
    };

    window.AddTaskFromUI = async () => {
        let selected = state.images.filter(img => img.Selected);
        if (selected.length === 0 && state.images.length === 1) {
            selected = [state.images[0]];
        }
        if (selected.length === 0 && state.selectedImageID) {
            const img = state.images.find(i => i.ID == state.selectedImageID);
            if (img) selected.push(img);
        }

        if (selected.length === 0) {
            addLog("Error: Select images (checkbox) first!");
            return;
        }

        const imgIDs = selected.map(i => i.ID).join("+");
        const paths = selected.map(i => i.FullPath).join("|");
        const tier = document.getElementById('tier-select').value;
        const ratio = document.getElementById('ratio-select').value;
        const prompt = document.getElementById('prompt').value || state.config.default_prompt;
        const negPrompt = document.getElementById('neg-prompt').value || state.config.default_neg_prompt;

        const parts = tier.split(" ");
        const agent = parts.slice(0, -1).join(" ");
        const size = parts[parts.length - 1];

        await AddTask(imgIDs, agent, size, ratio, prompt, negPrompt, paths);
        addLog("Task added for: " + imgIDs);
    };

    window.UpdateTaskFromUI = async () => {
        if (!state.selectedTaskID) return;
        const task = state.tasks.find(t => t.ID === state.selectedTaskID);
        if (!task) return;

        task.ImgIDs = document.getElementById('source-ids').value;
        task.Prompt = document.getElementById('prompt').value;
        task.NegativePrompt = document.getElementById('neg-prompt').value;
        const tier = document.getElementById('tier-select').value;
        const parts = tier.split(" ");
        task.Agent = parts.slice(0, -1).join(" ");
        task.Size = parts[parts.length - 1];
        task.Ratio = document.getElementById('ratio-select').value;

        await updateCostDisplay(task.Agent, task.Size);
        await UpdateTask(task);
    };

    window.SaveSettings = async () => {
        const c = state.config;
        c.api_key = document.getElementById('settings-api-key').value;
        c.output_dir = document.getElementById('settings-output-dir').value;
        c.debug = document.getElementById('settings-debug').checked;
        c.encourage_gen = document.getElementById('settings-encourage-gen').value;
        c.encourage_edt = document.getElementById('settings-encourage-edt').value;
        c.temperature = parseFloat(document.getElementById('settings-temp').value);
        c.top_p = parseFloat(document.getElementById('settings-top-p').value);
        c.top_k = parseInt(document.getElementById('settings-top-k').value);
        c.max_output_tokens = parseInt(document.getElementById('settings-max-tokens').value);

        c.safety_settings[0].threshold = document.getElementById('safety-harassment').value;
        c.safety_settings[1].threshold = document.getElementById('safety-hate').value;
        c.safety_settings[2].threshold = document.getElementById('safety-sex').value;
        c.safety_settings[3].threshold = document.getElementById('safety-danger').value;

        c.model_nano_flash = document.getElementById('settings-model-flash').value;
        c.model_nano_pro = document.getElementById('settings-model-pro').value;
        c.model_nano_2 = document.getElementById('settings-model-2').value;
        c.model_imagen = document.getElementById('settings-model-imagen').value;
        c.model_imagen_ultra = document.getElementById('settings-model-ultra').value;

        await SaveConfig(c);
        addLog("Configuration saved to config.json");
        const msg = document.getElementById('settings-saved-msg');
        if (msg) {
            msg.style.opacity = '1';
            setTimeout(() => {
                msg.style.opacity = '0';
            }, 2000);
        }
    };


    // Global click to hide context menu
    document.addEventListener('click', () => {
        const menu = document.getElementById('context-menu');
        if (menu) menu.style.display = 'none';
    });
}

function populateSettings(c) {
    if (!c) return;
    document.getElementById('settings-api-key').value = c.api_key || '';
    document.getElementById('settings-output-dir').value = c.output_dir || '';
    document.getElementById('settings-debug').checked = !!c.debug;
    document.getElementById('settings-encourage-gen').value = c.encourage_gen || '';
    document.getElementById('settings-encourage-edt').value = c.encourage_edt || '';
    document.getElementById('settings-temp').value = c.temperature || 1.0;
    document.getElementById('settings-top-p').value = c.top_p || 0.9;
    document.getElementById('settings-top-k').value = c.top_k || 40;
    document.getElementById('settings-max-tokens').value = c.max_output_tokens || 8192;

    if (c.safety_settings && c.safety_settings.length >= 4) {
        document.getElementById('safety-harassment').value = c.safety_settings[0].threshold;
        document.getElementById('safety-hate').value = c.safety_settings[1].threshold;
        document.getElementById('safety-sex').value = c.safety_settings[2].threshold;
        document.getElementById('safety-danger').value = c.safety_settings[3].threshold;
    }

    document.getElementById('settings-model-flash').value = c.model_nano_flash || '';
    document.getElementById('settings-model-pro').value = c.model_nano_pro || '';
    document.getElementById('settings-model-2').value = c.model_nano_2 || '';
    document.getElementById('settings-model-imagen').value = c.model_imagen || '';
    document.getElementById('settings-model-ultra').value = c.model_imagen_ultra || '';
}

function updateRunButtons() {
    const immBtn = document.getElementById('btn-run-immediate');
    const batchBtn = document.getElementById('btn-run-batch');

    if (!immBtn || !batchBtn) return;

    const allEnabledTasks = state.tasks.filter(t => !t.Disabled);
    const hasTasks = allEnabledTasks.length > 0;
    const runningTasks = state.tasks.filter(tk => (tk.RunningCount || 0) > 0);
    const totalRunning = runningTasks.length;

    // Immediate mode logic
    let immLabel = "RUN IMMEDIATE";
    let immDisabled = !hasTasks;

    if (allEnabledTasks.length === 1) {
        const t = allEnabledTasks[0];
        const runningCount = t.RunningCount || 0;
        immLabel = `RUN IMMEDIATE (${runningCount}/2)`;
        if (runningCount >= 2) {
            immDisabled = true;
        }
    } else if (allEnabledTasks.length > 1) {
        // If there are multiple enabled tasks, only do one at a time.
        if (totalRunning >= 1) {
            immDisabled = true;
        }
    }

    // Immediate: only works if there's a prompt. If Imagen, must have no source images OR only "GENERATE" images.
    const canImmediate = hasTasks && allEnabledTasks.every(t => {
        const hasPrompt = t.Prompt && t.Prompt.trim() !== "";
        const isImagen = t.Agent.includes("Imagen");

        // Find image objects for IDs in t.ImgIDs
        const ids = (t.ImgIDs || "").split("+").map(id => id.trim()).filter(id => id !== "");
        const taskImages = state.images.filter(img => ids.includes(img.ID));
        const onlyGenerate = taskImages.length > 0 && taskImages.every(img => img.FullPath === "<GENERATE>");
        const noImages = ids.length === 0;

        if (isImagen && !(noImages || onlyGenerate)) return false;
        return hasPrompt;
    });
    immBtn.disabled = immDisabled || !canImmediate;
    immBtn.innerText = immLabel;

    // Batch: all tasks must have same agent and not be Imagen
    let canBatch = hasTasks;
    if (canBatch) {
        const firstAgent = allEnabledTasks[0].Agent;
        if (firstAgent.includes("Imagen")) {
            canBatch = false;
        } else {
            canBatch = allEnabledTasks.every(t => {
                const hasPrompt = t.Prompt && t.Prompt.trim() !== "";
                return t.Agent === firstAgent && hasPrompt;
            });
        }
    }
    batchBtn.disabled = !canBatch || state.isRunning || totalRunning > 0;
}

async function showPreview(id) {
    const img = state.images.find(i => i.ID == id);
    if (!img || img.FullPath === "" || img.FullPath === "<GENERATE>") return;

    state.isHoveringImage = true;
    document.getElementById('editor-container').style.display = 'none';
    document.getElementById('preview-container').style.display = 'flex';

    const preview = document.getElementById('image-preview');
    try {
        const b64 = await GetImageBase64(img.FullPath);
        if (b64) {
            preview.innerHTML = `<img src="data:image/jpeg;base64,${b64}" class="preview-image">`;
        } else {
            preview.innerText = 'Error loading image data';
        }
    } catch (err) {
        preview.innerText = 'Error: ' + err;
    }
}

function hidePreview() {
    state.isHoveringImage = false;
    if (!state.selectedImageID) {
        document.getElementById('editor-container').style.display = 'block';
        document.getElementById('preview-container').style.display = 'none';
    }
}

function renderAll() {
    if (state.activeTab === 'create') {
        renderImageList();
        renderTaskList();
    } else if (state.activeTab === 'batches') {
        renderBatchList();
    }
}

// --- Image List ---

function renderImageList() {
    const list = document.getElementById('image-list');
    if (!list) return;
    list.innerHTML = '';

    state.images.forEach(img => {
        const item = document.createElement('div');
        item.className = 'list-item' + (state.selectedImageID === img.ID ? ' selected' : '');
        item.innerHTML = `
            <span style="width: 30px"><input type="checkbox" ${img.Selected ? 'checked' : ''} class="img-check"></span>
            <span style="width: 30px">${img.ID}</span>
            <span style="width: 60px">${img.SizeMB.toFixed(2)}</span>
            <span style="width: 50px">${img.TaskCount}</span>
            <span style="flex: 1; overflow: hidden; text-overflow: ellipsis; white-space: nowrap;">${img.FileName}</span>
        `;

        const check = item.querySelector('.img-check');
        check.onchange = (e) => {
            img.Selected = e.target.checked;
            updateRunButtons();
        };

        item.onmouseenter = () => showPreview(img.ID);
        item.onmouseleave = () => hidePreview();

        item.onclick = (e) => {
            if (e.target.type !== 'checkbox') {
                state.selectedImageID = (state.selectedImageID === img.ID ? null : img.ID);
                state.selectedTaskID = null;
                renderImageList();
                renderTaskList();
            }
        };

        item.oncontextmenu = (e) => {
            e.preventDefault();
            showContextMenu(e.clientX, e.clientY, [
                { label: 'Change Image', action: () => window.ChangeImageUI(img.ID) },
                { label: 'Delete Image', action: () => window.DeleteImage(img.ID) }
            ]);
        };

        list.appendChild(item);
    });
}

// --- Task List ---

function renderTaskList() {
    const list = document.getElementById('task-list');
    if (!list) return;
    list.innerHTML = '';

    state.tasks.forEach(task => {
        const item = document.createElement('div');
        item.className = 'list-item' + (state.selectedTaskID === task.ID ? ' selected' : '');
        if (task.Disabled) item.style.opacity = '0.5';

        item.innerHTML = `
            <span style="width: 60px">${task.ImgIDs}</span>
            <span style="width: 120px">${task.Agent} ${task.Size}</span>
            <span style="width: 60px">${task.Ratio}</span>
            <span style="width: 100px">${task.Status}</span>
            <span style="flex: 1; overflow: hidden; text-overflow: ellipsis; white-space: nowrap;">${task.Prompt}</span>
        `;

        item.onclick = () => {
            state.selectedTaskID = (state.selectedTaskID === task.ID ? null : task.ID);
            state.selectedImageID = null;

            if (state.selectedTaskID) {
                document.getElementById('editor-container').style.display = 'block';
                document.getElementById('preview-container').style.display = 'none';
                populateEditor(task);
            }

            renderImageList();
            renderTaskList();
        };

        item.oncontextmenu = async (e) => {
            e.preventDefault();
            const hasImg = await HasGeneratedImage(task.ID);
            const menuItems = [
                { label: task.Disabled ? 'Enable' : 'Disable', action: () => window.ToggleTaskDisabled(task.ID) }
            ];
            if (hasImg) {
                menuItems.push({ label: 'Show Generated Image', action: () => window.ShowGeneratedImage(task.ID) });
            }
            menuItems.push({ label: 'Duplicate Task', action: () => window.DuplicateTask(task.ID) });
            menuItems.push({ label: 'Delete Task', action: () => window.DeleteTask(task.ID) });

            showContextMenu(e.clientX, e.clientY, menuItems);
        };

        list.appendChild(item);
    });
}

async function populateEditor(task) {
    document.getElementById('source-ids').value = task.ImgIDs;
    document.getElementById('prompt').value = task.Prompt;
    document.getElementById('neg-prompt').value = task.NegativePrompt;
    document.getElementById('tier-select').value = task.Agent + " " + task.Size;
    document.getElementById('ratio-select').value = task.Ratio;
    await updateCostDisplay(task.Agent, task.Size);
}

async function updateCostDisplay(agent, size) {
    const costImm = await GetCost(agent, size, "Immediate");
    const costBatch = await GetCost(agent, size, "Batch");
    document.getElementById('cost-display').innerText = `Immediate: $${costImm.toFixed(4)} | Batch: $${costBatch.toFixed(4)}`;
}

// --- Batch List ---

function renderBatchList() {
    const list = document.getElementById('batch-list');
    if (!list) return;
    list.innerHTML = '';

    if (state.batchJobs.length === 0) {
        list.innerHTML = '<div style="padding: 20px; color: #888;">No active batch jobs.</div>';
        return;
    }

    state.batchJobs.forEach(job => {
        const item = document.createElement('div');
        item.className = 'batch-item';
        item.style = 'background-color: #151d29; border: 1px solid #34495e; padding: 15px; margin-bottom: 10px; border-radius: 8px;';

        const isFinished = ["SUCCEEDED", "FAILED", "CANCELLED", "EXPIRED", "Success", "Failed"].includes(job.Status);
        const progress = isFinished ? 100 : (job.Status === "Submitted" ? 10 : 50); // Dummy progress for now as API doesn't provide it clearly in percentage

        item.innerHTML = `
            <div style="display: flex; justify-content: space-between; margin-bottom: 8px;">
                <span style="font-weight: bold; color: #3498db;">${job.JobID}</span>
                <span style="color: ${isFinished ? (job.Status === 'SUCCEEDED' ? '#2ecc71' : '#e74c3c') : '#f1c40f'}">${job.Status}</span>
            </div>
            <div style="font-size: 0.85em; color: #888; margin-bottom: 10px;">Submitted at: ${new Date(job.SubmittedAt).toLocaleString()}</div>
            <div class="progress-bar-bg" style="background-color: #2c3e50; height: 10px; border-radius: 5px; overflow: hidden;">
                <div class="progress-bar-fill" style="background-color: #3498db; width: ${progress}%; height: 100%; transition: width 0.3s;"></div>
            </div>
        `;
        list.appendChild(item);
    });
}

// --- Context Menu ---

function showContextMenu(x, y, items) {
    let menu = document.getElementById('context-menu');
    if (!menu) {
        menu = document.createElement('div');
        menu.id = 'context-menu';
        menu.className = 'context-menu';
        document.body.appendChild(menu);
    }

    menu.innerHTML = '';
    items.forEach(item => {
        const div = document.createElement('div');
        div.innerText = item.label;
        div.onclick = item.action;
        menu.appendChild(div);
    });

    menu.style.left = x + 'px';
    menu.style.top = y + 'px';
    menu.style.display = 'flex';
}

// --- Logs ---

function addLog(msg) {
    const logArea = document.getElementById('logs');
    if (!logArea) return;
    const entry = document.createElement('div');
    const time = new Date().toLocaleTimeString();
    entry.innerText = `[${time}] ${msg}`;
    logArea.appendChild(entry);
    logArea.scrollTop = logArea.scrollHeight;
}
