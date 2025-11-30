package main

import (
	"bytes"
	"encoding/json"
	"fmt"
	"image/color"
	"io"
	"net/http"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/app"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/dialog"
	"fyne.io/fyne/v2/widget"
)

type UIForum struct {
	ID     uint   `json:"ID"`
	GameID uint   `json:"GameID"`
	Name   string `json:"Name"`
}

type UIPost struct {
	ID        uint     `json:"ID"`
	ForumID   uint     `json:"ForumID"`
	UserID    uint     `json:"UserID"`
	Title     string   `json:"Title"`
	Content   string   `json:"Content"`
	CreatedAt string   `json:"CreatedAt"`
	User      UIUser   `json:"User"`
}

type UIUser struct {
	ID       uint   `json:"ID"`
	Username string `json:"Username"`
	Email    string `json:"Email"`
}

type UIComment struct {
	ID        uint   `json:"ID"`
	PostID    uint   `json:"PostID"`
	UserID    uint   `json:"UserID"`
	Content   string `json:"Content"`
	CreatedAt string `json:"CreatedAt"`
	User      UIUser `json:"User"`
}

var (
	baseURL = "http://localhost:8082"
	token   string
	currentUser UIUser
)

func main() {
	a := app.New()
	w := a.NewWindow("Persona 5 Forum")
	w.Resize(fyne.NewSize(1200, 800))

	showLoginScreen(w)
	w.ShowAndRun()
}


func showLoginScreen(w fyne.Window) {
	usernameEntry := widget.NewEntry()
	usernameEntry.SetPlaceHolder("Username")

	passwordEntry := widget.NewPasswordEntry()
	passwordEntry.SetPlaceHolder("Password")

	emailEntry := widget.NewEntry()
	emailEntry.SetPlaceHolder("Email (for register)")

	isRegister := false

	form := &widget.Form{
		Items: []*widget.FormItem{
			{Text: "Username:", Widget: usernameEntry},
			{Text: "Password:", Widget: passwordEntry},
			{Text: "Email:", Widget: emailEntry},
		},
		OnSubmit: func() {
			if isRegister {
				register(usernameEntry.Text, passwordEntry.Text, emailEntry.Text, w)
			} else {
				login(usernameEntry.Text, passwordEntry.Text, w)
			}
		},
	}

	toggleBtn := widget.NewButton("Switch to Register", func() {
		isRegister = !isRegister
		if isRegister {
			toggleBtn.SetText("Switch to Login")
			form.SubmitText = "Register"
		} else {
			toggleBtn.SetText("Switch to Register")
			form.SubmitText = "Login"
		}
		form.Refresh()
	})

	content := container.NewVBox(
		widget.NewLabelWithStyle("Persona 5 Forum", fyne.TextAlignCenter, fyne.TextStyle{Bold: true}),
		form,
		toggleBtn,
	)

	w.SetContent(content)
}

func register(username, password, email string, w fyne.Window) {
	data := map[string]string{
		"username": username,
		"password": password,
		"email":    email,
	}
	jsonData, _ := json.Marshal(data)

	resp, err := http.Post(baseURL+"/register", "application/json", bytes.NewBuffer(jsonData))
	if err != nil {
		dialog.ShowError(err, w)
		return
	}
	defer resp.Body.Close()

	if resp.StatusCode == 200 {
		dialog.ShowInformation("Success", "Registration successful", w)
	} else {
		body, _ := io.ReadAll(resp.Body)
		dialog.ShowError(fmt.Errorf("Registration failed: %s", string(body)), w)
	}
}

func login(username, password string, w fyne.Window) {
	data := map[string]string{
		"username": username,
		"password": password,
	}
	jsonData, _ := json.Marshal(data)

	resp, err := http.Post(baseURL+"/login", "application/json", bytes.NewBuffer(jsonData))
	if err != nil {
		dialog.ShowError(err, w)
		return
	}
	defer resp.Body.Close()

	if resp.StatusCode == 200 {
		var result map[string]string
		json.NewDecoder(resp.Body).Decode(&result)
		token = result["token"]
		// Assume user ID from token, but for simplicity, set dummy
		currentUser = UIUser{ID: 1, Username: username}
		showMainScreen(w)
	} else {
		body, _ := io.ReadAll(resp.Body)
		dialog.ShowError(fmt.Errorf("Login failed: %s", string(body)), w)
	}
}

func showMainScreen(w fyne.Window) {
	// Top bar with user info and logout
	userLabel := widget.NewLabel(fmt.Sprintf("Logged in as: %s", currentUser.Username))
	logoutBtn := widget.NewButton("Logout", func() {
		token = ""
		showLoginScreen(w)
	})

	topBar := container.NewHBox(userLabel, logoutBtn)

	// Forums list
	forumsList := widget.NewList(
		func() int { return len(forums) },
		func() fyne.CanvasObject {
			return widget.NewLabel("Template")
		},
		func(id widget.ListItemID, obj fyne.CanvasObject) {
			obj.(*widget.Label).SetText(forums[id].Name)
		},
	)
	forumsList.OnSelected = func(id widget.ListItemID) {
		loadPosts(forums[id].ID, w)
	}

	// Posts list
	postsList := widget.NewList(
		func() int { return len(posts) },
		func() fyne.CanvasObject {
			return container.NewVBox(widget.NewLabel("Title"), widget.NewLabel("Author"))
		},
		func(id widget.ListItemID, obj fyne.CanvasObject) {
			container := obj.(*container.VBox)
			container.Objects[0].(*widget.Label).SetText(posts[id].Title)
			container.Objects[1].(*widget.Label).SetText(posts[id].User.Username)
		},
	)
	postsList.OnSelected = func(id widget.ListItemID) {
		showPostDetails(posts[id], w)
	}

	// Post details
	postTitleLabel := widget.NewLabel("")
	postContentLabel := widget.NewLabel("")
	commentsList := widget.NewList(
		func() int { return len(comments) },
		func() fyne.CanvasObject {
			return container.NewVBox(widget.NewLabel("Comment"), widget.NewLabel("Author"))
		},
		func(id widget.ListItemID, obj fyne.CanvasObject) {
			container := obj.(*container.VBox)
			container.Objects[0].(*widget.Label).SetText(comments[id].Content)
			container.Objects[1].(*widget.Label).SetText(comments[id].User.Username)
		},
	)

	commentEntry := widget.NewMultiLineEntry()
	commentEntry.SetPlaceHolder("Add a comment...")
	submitCommentBtn := widget.NewButton("Submit Comment", func() {
		if selectedPost.ID != 0 {
			addComment(selectedPost.ID, commentEntry.Text, w)
			commentEntry.SetText("")
		}
	})

	postDetails := container.NewVBox(
		postTitleLabel,
		postContentLabel,
		widget.NewLabel("Comments:"),
		commentsList,
		commentEntry,
		submitCommentBtn,
	)

	// Layout
	leftPanel := container.NewVBox(widget.NewLabel("Forums"), forumsList)
	centerPanel := container.NewVBox(widget.NewLabel("Posts"), postsList)
	rightPanel := container.NewScroll(postDetails)

	split1 := container.NewHSplit(leftPanel, centerPanel)
	split1.SetOffset(0.3)
	split2 := container.NewHSplit(split1, rightPanel)
	split2.SetOffset(0.7)

	content := container.NewBorder(topBar, nil, nil, nil, split2)
	w.SetContent(content)

	loadForums(w)
}

var forums []UIForum
var posts []UIPost
var comments []UIComment
var selectedPost UIPost

func loadForums(w fyne.Window) {
	resp, err := http.Get(baseURL + "/forums")
	if err != nil {
		dialog.ShowError(err, w)
		return
	}
	defer resp.Body.Close()

	json.NewDecoder(resp.Body).Decode(&forums)
	// Refresh list somehow, but for simplicity, assume static
}

func loadPosts(forumID uint, w fyne.Window) {
	resp, err := http.Get(fmt.Sprintf("%s/forums/%d/posts", baseURL, forumID))
	if err != nil {
		dialog.ShowError(err, w)
		return
	}
	defer resp.Body.Close()

	json.NewDecoder(resp.Body).Decode(&posts)
	// Refresh
}

func showPostDetails(post UIPost, w fyne.Window) {
	selectedPost = post
	// Set labels
	// Load comments
	loadComments(post.ID, w)
}

func loadComments(postID uint, w fyne.Window) {
	resp, err := http.Get(fmt.Sprintf("%s/posts/%d/comments", baseURL, postID))
	if err != nil {
		dialog.ShowError(err, w)
		return
	}
	defer resp.Body.Close()

	json.NewDecoder(resp.Body).Decode(&comments)
}

func addComment(postID uint, content string, w fyne.Window) {
	data := map[string]string{"content": content}
	jsonData, _ := json.Marshal(data)

	req, _ := http.NewRequest("POST", fmt.Sprintf("%s/posts/%d/comments", baseURL, postID), bytes.NewBuffer(jsonData))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+token)

	client := &http.Client{}
	resp, err := client.Do(req)
	if err != nil {
		dialog.ShowError(err, w)
		return
	}
	defer resp.Body.Close()

	if resp.StatusCode == 201 {
		loadComments(postID, w)
	} else {
		body, _ := io.ReadAll(resp.Body)
		dialog.ShowError(fmt.Errorf("Failed to add comment: %s", string(body)), w)
	}
}