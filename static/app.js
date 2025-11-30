document.addEventListener('DOMContentLoaded', function() {
    const token = localStorage.getItem('token');
    if (token) {
        showMainSection();
        loadForums();
    } else {
        showLoginSection();
    }

    // Event listeners
    document.getElementById('login-btn').addEventListener('click', login);
    document.getElementById('register-btn').addEventListener('click', register);
    document.getElementById('show-register').addEventListener('click', () => toggleAuthForms('register'));
    document.getElementById('show-login').addEventListener('click', () => toggleAuthForms('login'));
    document.getElementById('logout-btn').addEventListener('click', logout);
    document.getElementById('submit-post-btn').addEventListener('click', createPost);
    document.getElementById('submit-comment-btn').addEventListener('click', createComment);
});

function showLoginSection() {
    document.getElementById('login-section').classList.remove('hidden');
    document.getElementById('main-section').classList.add('hidden');
}

function showMainSection() {
    document.getElementById('login-section').classList.add('hidden');
    document.getElementById('main-section').classList.remove('hidden');
    const token = localStorage.getItem('token');
    if (token) {
        // Decode token to get username (simplified)
        const payload = JSON.parse(atob(token.split('.')[1]));
        document.getElementById('username-display').textContent = `Welcome, ${payload.user_id}`;
    }
}

function toggleAuthForms(form) {
    const loginForm = document.getElementById('login-form');
    const registerForm = document.getElementById('register-form');
    if (form === 'register') {
        loginForm.classList.add('hidden');
        registerForm.classList.remove('hidden');
    } else {
        registerForm.classList.add('hidden');
        loginForm.classList.remove('hidden');
    }
}

async function login() {
    const username = document.getElementById('login-username').value;
    const password = document.getElementById('login-password').value;

    try {
        const response = await fetch('/login', {
            method: 'POST',
            headers: { 'Content-Type': 'application/json' },
            body: JSON.stringify({ username, password })
        });

        if (response.ok) {
            const data = await response.json();
            localStorage.setItem('token', data.token);
            showMainSection();
            loadForums();
        } else {
            alert('Login failed');
        }
    } catch (error) {
        console.error('Error:', error);
        alert('Login failed');
    }
}

async function register() {
    const username = document.getElementById('reg-username').value;
    const email = document.getElementById('reg-email').value;
    const password = document.getElementById('reg-password').value;

    try {
        const response = await fetch('/register', {
            method: 'POST',
            headers: { 'Content-Type': 'application/json' },
            body: JSON.stringify({ username, password, email })
        });

        if (response.ok) {
            alert('Registration successful. Please login.');
            toggleAuthForms('login');
        } else {
            alert('Registration failed');
        }
    } catch (error) {
        console.error('Error:', error);
        alert('Registration failed');
    }
}

function logout() {
    localStorage.removeItem('token');
    showLoginSection();
}

async function loadForums() {
    try {
        const response = await fetch('/forums');
        if (response.ok) {
            const forums = await response.json();
            const forumsUl = document.getElementById('forums-ul');
            forumsUl.innerHTML = '';
            forums.forEach(forum => {
                const li = document.createElement('li');
                li.textContent = forum.Name;
                li.addEventListener('click', () => loadPosts(forum.ID, forum.Name));
                forumsUl.appendChild(li);
            });
        }
    } catch (error) {
        console.error('Error loading forums:', error);
    }
}

async function loadPosts(forumId, forumName) {
    document.getElementById('forum-title').textContent = forumName;
    document.getElementById('new-post-form').classList.remove('hidden');

    try {
        const response = await fetch(`/forums/${forumId}/posts`);
        if (response.ok) {
            const posts = await response.json();
            const postsList = document.getElementById('posts-list');
            postsList.innerHTML = '';
            posts.forEach(post => {
                const postDiv = document.createElement('div');
                postDiv.className = 'post-item';
                postDiv.innerHTML = `
                    <div class="post-title">${post.Title}</div>
                    <div class="post-author">by ${post.User.Username}</div>
                    <div>${post.Content.substring(0, 100)}...</div>
                `;
                postDiv.addEventListener('click', () => showPostDetails(post));
                postsList.appendChild(postDiv);
            });
        }
    } catch (error) {
        console.error('Error loading posts:', error);
    }
}

function showPostDetails(post) {
    const postContent = document.getElementById('post-content');
    postContent.innerHTML = `
        <h3>${post.Title}</h3>
        <p>by ${post.User.Username}</p>
        <p>${post.Content}</p>
    `;

    loadComments(post.ID);
}

async function loadComments(postId) {
    try {
        const response = await fetch(`/posts/${postId}/comments`);
        if (response.ok) {
            const comments = await response.json();
            const commentsList = document.getElementById('comments-list');
            commentsList.innerHTML = '';
            comments.forEach(comment => {
                const li = document.createElement('li');
                li.className = 'comment-item';
                li.innerHTML = `
                    <div>${comment.Content}</div>
                    <div class="comment-author">by ${comment.User.Username}</div>
                `;
                commentsList.appendChild(li);
            });
        }
    } catch (error) {
        console.error('Error loading comments:', error);
    }
}

async function createPost() {
    const title = document.getElementById('post-title').value;
    const content = document.getElementById('post-content').value;
    const forumTitle = document.getElementById('forum-title').textContent;
    const forumId = getForumIdFromTitle(forumTitle); // Need to implement

    const token = localStorage.getItem('token');

    try {
        const response = await fetch(`/forums/${forumId}/posts`, {
            method: 'POST',
            headers: {
                'Content-Type': 'application/json',
                'Authorization': `Bearer ${token}`
            },
            body: JSON.stringify({ title, content })
        });

        if (response.ok) {
            document.getElementById('post-title').value = '';
            document.getElementById('post-content').value = '';
            loadPosts(forumId, forumTitle);
        } else {
            alert('Failed to create post');
        }
    } catch (error) {
        console.error('Error:', error);
        alert('Failed to create post');
    }
}

async function createComment() {
    const content = document.getElementById('comment-content').value;
    // Assume current post ID is stored
    const postId = 1; // Placeholder

    const token = localStorage.getItem('token');

    try {
        const response = await fetch(`/posts/${postId}/comments`, {
            method: 'POST',
            headers: {
                'Content-Type': 'application/json',
                'Authorization': `Bearer ${token}`
            },
            body: JSON.stringify({ content })
        });

        if (response.ok) {
            document.getElementById('comment-content').value = '';
            loadComments(postId);
        } else {
            alert('Failed to create comment');
        }
    } catch (error) {
        console.error('Error:', error);
        alert('Failed to create comment');
    }
}

// Helper function to get forum ID (simplified)
function getForumIdFromTitle(title) {
    // This is a placeholder; in a real app, you'd store the current forum ID
    return 1; // Assuming first forum
}