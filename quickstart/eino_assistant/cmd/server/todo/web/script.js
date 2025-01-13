// URL 参数处理
function getQueryParams() {
    const params = new URLSearchParams(window.location.search);
    return {
        query: params.get('q') || '',
        is_done: params.get('done') === null ? null : params.get('done') === 'true',
        limit: parseInt(params.get('limit')) || 10,
        sort: params.get('sort') || 'urgency'
    };
}

function updateQueryParams(params) {
    const url = new URL(window.location);
    if (params.query) url.searchParams.set('q', params.query);
    else url.searchParams.delete('q');
    
    if (params.is_done !== null) url.searchParams.set('done', params.is_done);
    else url.searchParams.delete('done');
    
    if (params.limit !== 10) url.searchParams.set('limit', params.limit);
    else url.searchParams.delete('limit');
    
    if (params.sort !== 'urgency') url.searchParams.set('sort', params.sort);
    else url.searchParams.delete('sort');
    
    window.history.pushState({}, '', url);
}

// 表单初始化
function initializeFormValues() {
    const params = getQueryParams();
    document.getElementById('searchInput').value = params.query;
    document.getElementById('statusFilter').value = params.is_done === null ? '' : params.is_done;
    document.getElementById('limitFilter').value = params.limit;
    document.getElementById('sortFilter').value = params.sort;
}

// 时间处理函数
function formatDate(dateStr) {
    if (!dateStr) return '';
    const date = new Date(dateStr);
    return date.toLocaleString('zh-CN', {
        year: 'numeric',
        month: '2-digit',
        day: '2-digit',
        hour: '2-digit',
        minute: '2-digit'
    });
}

function calculateUrgency(todo) {
    if (!todo.deadline || todo.completed) return Infinity;
    const now = new Date();
    const deadline = new Date(todo.deadline);
    const hoursLeft = (deadline - now) / (1000 * 60 * 60);
    return hoursLeft;
}

function formatTimeDiff(diffMs) {
    const hours = Math.floor(Math.abs(diffMs) / (1000 * 60 * 60));
    const minutes = Math.floor((Math.abs(diffMs) % (1000 * 60 * 60)) / (1000 * 60));
    const seconds = Math.floor((Math.abs(diffMs) % (1000 * 60)) / 1000);

    if (hours > 2) {
        return `${hours} 小时`;
    } else if (hours > 0) {
        return `${hours} 小时 ${minutes} 分钟`;
    } else if (minutes > 0) {
        return `${minutes} 分钟 ${seconds} 秒`;
    } else {
        return `${seconds} 秒`;
    }
}

function getDeadlineStatus(deadline, completed) {
    if (!deadline) return null;
    if (completed) return { status: 'completed', class: 'text-gray-500' };
    
    const now = new Date();
    const deadlineDate = new Date(deadline);
    const hoursLeft = (deadlineDate - now) / (1000 * 60 * 60);
    
    if (hoursLeft < 0) return { status: 'error', class: 'text-red-600 font-bold' };
    if (hoursLeft <= 12) return { status: 'warning', class: 'text-yellow-600 font-bold' };
    return { status: 'normal', class: 'text-gray-500' };
}

// 倒计时处理
function updateCountdown(element, deadline) {
    const now = new Date();
    const diffMs = deadline - now;
    
    if (diffMs <= 0) {
        loadTodos();
        return;
    }

    let text = `截止: ${formatDate(deadline)}`;
    if (diffMs > 0) {
        text += ` (剩余 ${formatTimeDiff(diffMs)})`;
    } else {
        text += ` (已超出 ${formatTimeDiff(Math.abs(diffMs))})`;
    }
    
    element.textContent = text;
}

// Todo 列表处理
async function loadTodos() {
    const params = getQueryParams();
    try {
        const response = await fetch('/todo/api', {
            method: 'POST',
            headers: { 'Content-Type': 'application/json' },
            body: JSON.stringify({
                action: 'list',
                list: params
            })
        });
        const data = await response.json();
        if (data.status === 'success') {
            renderTodos(data.todo_list);
        }
    } catch (error) {
        console.error('Failed to load todos:', error);
    }
}

function renderTodos(todos) {
    const list = document.getElementById('todoList');
    const template = document.getElementById('todoTemplate');
    list.innerHTML = '';

    // 清除所有现有的倒计时定时器
    if (window.countdownTimers) {
        window.countdownTimers.forEach(timer => clearInterval(timer));
    }
    window.countdownTimers = [];

    // 排序
    const sortType = document.getElementById('sortFilter').value;
    todos.sort((a, b) => {
        if (sortType === 'urgency') {
            return calculateUrgency(a) - calculateUrgency(b);
        } else if (sortType === 'deadline') {
            if (!a.deadline) return 1;
            if (!b.deadline) return -1;
            return new Date(a.deadline) - new Date(b.deadline);
        } else { // created
            return new Date(b.created_at) - new Date(a.created_at);
        }
    });

    todos.forEach(todo => {
        const clone = template.content.cloneNode(true);
        const item = clone.querySelector('.todo-item');
        
        item.dataset.id = todo.id;
        item.dataset.todo = JSON.stringify(todo);
        item.querySelector('.todo-checkbox').checked = todo.completed;
        item.querySelector('.todo-title').textContent = todo.title;
        item.querySelector('.todo-content').textContent = todo.content;
        item.querySelector('.todo-created').textContent = `创建: ${formatDate(todo.created_at)}`;
        
        if (todo.deadline) {
            const deadlineEl = item.querySelector('.todo-deadline');
            const status = getDeadlineStatus(todo.deadline, todo.completed);
            deadlineEl.className = `todo-deadline ${status.class}`;

            const now = new Date();
            const deadline = new Date(todo.deadline);
            const hoursLeft = (deadline - now) / (1000 * 60 * 60);
            const needsCountdown = !todo.completed && hoursLeft > 0 && hoursLeft <= 2;

            if (needsCountdown) {
                updateCountdown(deadlineEl, deadline);
                const timer = setInterval(() => {
                    updateCountdown(deadlineEl, deadline);
                }, 1000);
                window.countdownTimers.push(timer);
            } else {
                const diffMs = deadline - now;
                let text = `截止: ${formatDate(todo.deadline)}`;
                if (!todo.completed) {
                    text += ` (${diffMs > 0 ? '剩余 ' : '已超出 '}${formatTimeDiff(diffMs)})`;
                }
                deadlineEl.textContent = text;
            }
        }

        if (todo.completed) {
            item.querySelector('.todo-title').classList.add('line-through', 'text-gray-500');
            item.querySelector('.todo-content').classList.add('line-through', 'text-gray-500');
        }

        list.appendChild(clone);
    });
}

// 对话框处理
function openAddDialog() {
    document.getElementById('addDialog').classList.remove('hidden');
    document.getElementById('addForm').reset();
}

function closeAddDialog() {
    document.getElementById('addDialog').classList.add('hidden');
}

function openEditDialog(todo) {
    const dialog = document.getElementById('editDialog');
    const form = document.getElementById('editForm');
    
    form.id.value = todo.id;
    form.title.value = todo.title;
    form.content.value = todo.content;
    form.deadline.value = todo.deadline ? todo.deadline.slice(0, 16) : '';
    
    dialog.classList.remove('hidden');
}

function closeEditDialog() {
    document.getElementById('editDialog').classList.add('hidden');
}

// 事件处理
function debounce(func, delay) {
    let timer;
    return function() {
        clearTimeout(timer);
        timer = setTimeout(() => func.apply(this, arguments), delay);
    }
}

const handleFilterChange = debounce(() => {
    const params = {
        query: document.getElementById('searchInput').value,
        is_done: document.getElementById('statusFilter').value === '' ? null : document.getElementById('statusFilter').value === 'true',
        limit: parseInt(document.getElementById('limitFilter').value),
        sort: document.getElementById('sortFilter').value
    };
    updateQueryParams(params);
    loadTodos();
}, 300);

// 在文件顶部添加定时器变量
let autoRefreshTimer;

// 初始化事件监听
document.addEventListener('DOMContentLoaded', () => {
    // 添加任务
    document.getElementById('addTodoBtn').addEventListener('click', openAddDialog);
    document.getElementById('addForm').addEventListener('submit', async (e) => {
        e.preventDefault();
        const form = e.target;
        const todo = {
            title: form.title.value,
            content: form.content.value,
            deadline: form.deadline.value
        };

        try {
            const response = await fetch('/todo/api', {
                method: 'POST',
                headers: { 'Content-Type': 'application/json' },
                body: JSON.stringify({
                    action: 'add',
                    todo: todo
                })
            });
            const data = await response.json();
            if (data.status === 'success') {
                closeAddDialog();
                form.reset();
                loadTodos();
            }
        } catch (error) {
            console.error('Failed to add todo:', error);
        }
    });

    // 更新任务状态
    document.getElementById('todoList').addEventListener('change', async (e) => {
        if (e.target.matches('.todo-checkbox')) {
            const item = e.target.closest('.todo-item');
            const id = item.dataset.id;
            const completed = e.target.checked;

            try {
                const response = await fetch('/todo/api', {
                    method: 'POST',
                    headers: { 'Content-Type': 'application/json' },
                    body: JSON.stringify({
                        action: 'update',
                        todo: { id, completed }
                    })
                });
                const data = await response.json();
                if (data.status === 'success') {
                    loadTodos();
                }
            } catch (error) {
                console.error('Failed to update todo:', error);
            }
        }
    });

    // 删除任务
    document.getElementById('todoList').addEventListener('click', async (e) => {
        if (e.target.closest('.delete-btn')) {
            const item = e.target.closest('.todo-item');
            const id = item.dataset.id;

            if (!confirm('确定要删除这个任务吗？')) return;

            try {
                const response = await fetch('/todo/api', {
                    method: 'POST',
                    headers: { 'Content-Type': 'application/json' },
                    body: JSON.stringify({
                        action: 'delete',
                        todo: { id }
                    })
                });
                const data = await response.json();
                if (data.status === 'success') {
                    loadTodos();
                }
            } catch (error) {
                console.error('Failed to delete todo:', error);
            }
        }
    });

    // 编辑任务
    document.getElementById('todoList').addEventListener('click', async (e) => {
        if (e.target.closest('.edit-btn')) {
            const item = e.target.closest('.todo-item');
            const todo = JSON.parse(item.dataset.todo);
            openEditDialog(todo);
        }
    });

    document.getElementById('editForm').addEventListener('submit', async (e) => {
        e.preventDefault();
        const form = e.target;
        const todo = {
            id: form.id.value,
            title: form.title.value,
            content: form.content.value,
            deadline: form.deadline.value
        };

        try {
            const response = await fetch('/todo/api', {
                method: 'POST',
                headers: { 'Content-Type': 'application/json' },
                body: JSON.stringify({
                    action: 'update',
                    todo: todo
                })
            });
            const data = await response.json();
            if (data.status === 'success') {
                closeEditDialog();
                loadTodos();
            }
        } catch (error) {
            console.error('Failed to update todo:', error);
        }
    });

    // 筛选事件
    document.getElementById('searchInput').addEventListener('input', handleFilterChange);
    document.getElementById('statusFilter').addEventListener('change', handleFilterChange);
    document.getElementById('limitFilter').addEventListener('change', handleFilterChange);
    document.getElementById('sortFilter').addEventListener('change', handleFilterChange);

    // 浏览器前进/后退
    window.addEventListener('popstate', () => {
        initializeFormValues();
        loadTodos();
    });

    // 设置自动刷新定时器
    autoRefreshTimer = setInterval(loadTodos, 10000); // 每10秒刷新一次

    // 修改页面卸载事件处理，清理所有定时器
    window.addEventListener('beforeunload', () => {
        if (window.countdownTimers) {
            window.countdownTimers.forEach(timer => clearInterval(timer));
        }
        if (autoRefreshTimer) {
            clearInterval(autoRefreshTimer);
        }
    });

    // 初始化
    initializeFormValues();
    loadTodos();
}); 