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
    SaveSessionUI,
    LoadSessionUI,
    GetConfig,
    SaveConfig,
    TestConnection,
    GetCost
} from '../wailsjs/go/main/App';
import { EventsOn } from '../wailsjs/runtime/runtime';

let state = {
    images: [],
    tasks: [],
    config: {},
    activeTab: 'create',
    selectedImageID: null,
    selectedTaskID: null
};

// --- Initialization ---

window.addEventListener('DOMContentLoaded', async () => {
    try {
        state.config = await GetConfig();
        if (state.config) {
            document.getElementById('settings-api-key').value = state.config.api_key || '';
        }

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
}

function setupEventListeners() {
    EventsOn("log", (msg) => addLog(msg));
    EventsOn("images_updated", async () => {
        await refreshData();
        renderImageList();
    });
    EventsOn("tasks_updated", async () => {
        await refreshData();
        renderTaskList();
    });

    document.querySelectorAll('.tab').forEach(tab => {
        tab.addEventListener('click', () => {
            document.querySelectorAll('.tab').forEach(t => t.classList.remove('active'));
            tab.classList.add('active');
            state.activeTab = tab.dataset.tab;

            document.querySelectorAll('.tab-content').forEach(c => c.style.display = 'none');
            document.getElementById('tab-' + state.activeTab).style.display = (state.activeTab === 'create' ? 'flex' : 'block');

            renderAll();
        });
    });

    // Control Buttons
    document.getElementById('btn-load-session').onclick = () => LoadSessionUI();
    document.getElementById('btn-save-session').onclick = () => SaveSessionUI();
    document.getElementById('btn-run').onclick = () => RunTasks();

    // Global Exposure for HTML onclicks
    window.SelectAndAddMultipleImages = SelectAndAddMultipleImages;
    window.CreateNewImage = CreateNewImage;
    window.TestConnection = TestConnection;

    window.AddTaskFromUI = async () => {
        const selected = state.images.filter(img => img.Selected);
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
        const prompt = document.getElementById('prompt').value;
        const negPrompt = document.getElementById('neg-prompt').value;

        const parts = tier.split(" ");
        const agent = parts.slice(0, -1).join(" ");
        const size = parts[parts.length - 1];

        await AddTask(imgIDs, agent, size, ratio, prompt, negPrompt, paths);
    };

    window.SaveSettings = async () => {
        state.config.api_key = document.getElementById('settings-api-key').value;
        await SaveConfig(state.config);
        addLog("Settings saved");
    };
}

function renderAll() {
    if (state.activeTab === 'create') {
        renderImageList();
        renderTaskList();
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
        };

        item.onclick = (e) => {
            if (e.target.type !== 'checkbox') selectImage(img.ID);
        };
        list.appendChild(item);
    });
}

function selectImage(id) {
    state.selectedImageID = id;
    state.selectedTaskID = null;
    renderImageList();
    renderTaskList();

    const img = state.images.find(i => i.ID == id);
    const preview = document.getElementById('image-preview');
    if (img) {
        if (img.FullPath === "<GENERATE>") {
            preview.innerText = 'GENERATE';
        } else {
            preview.innerText = img.FullPath;
        }
    }
}

// --- Task List ---

function renderTaskList() {
    const list = document.getElementById('task-list');
    if (!list) return;
    list.innerHTML = '';

    state.tasks.forEach(task => {
        const item = document.createElement('div');
        item.className = 'list-item' + (state.selectedTaskID === task.ID ? ' selected' : '');
        item.innerHTML = `
            <span style="width: 60px">${task.ImgIDs}</span>
            <span style="width: 120px">${task.Agent} ${task.Size}</span>
            <span style="width: 60px">${task.Ratio}</span>
            <span style="width: 100px">${task.Status}</span>
            <span style="width: 80px">$${task.Cost.toFixed(4)}</span>
            <span style="flex: 1; overflow: hidden; text-overflow: ellipsis; white-space: nowrap;">${task.Prompt}</span>
        `;
        item.onclick = () => selectTask(task.ID);
        list.appendChild(item);
    });
}

function selectTask(id) {
    state.selectedTaskID = id;
    state.selectedImageID = null;
    renderImageList();
    renderTaskList();

    const task = state.tasks.find(t => t.ID === id);
    if (task) {
        document.getElementById('source-ids').value = task.ImgIDs;
        document.getElementById('prompt').value = task.Prompt;
        document.getElementById('neg-prompt').value = task.NegativePrompt;
        document.getElementById('tier-select').value = task.Agent + " " + task.Size;
        document.getElementById('ratio-select').value = task.Ratio;
        document.getElementById('cost-display').innerText = "Cost: $" + task.Cost.toFixed(4);
    }
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
