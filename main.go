package main

import (
	"bytes"
	"fmt"
	"io"
	"io/ioutil"
	"mime/multipart"
	"net/http"
	"os"
	"path/filepath"
	"sync"
	"time"

	"a21hc3NpZ25tZW50/client"
	"a21hc3NpZ25tZW50/db/filebased"
	"a21hc3NpZ25tZW50/handler/api"
	"a21hc3NpZ25tZW50/handler/web"
	"a21hc3NpZ25tZW50/middleware"
	repo "a21hc3NpZ25tZW50/repository"
	"a21hc3NpZ25tZW50/service"
	"embed"

	"github.com/gin-gonic/gin"
	_ "github.com/lib/pq"
)

type APIHandler struct {
	UserAPIHandler     api.UserAPI
	CategoryAPIHandler api.CategoryAPI
	TaskAPIHandler     api.TaskAPI
}

type ClientHandler struct {
	AuthWeb      web.AuthWeb
	HomeWeb      web.HomeWeb
	DashboardWeb web.DashboardWeb
	TaskWeb      web.TaskWeb
	CategoryWeb  web.CategoryWeb
	ModalWeb     web.ModalWeb
	ChatbotWeb   web.ChatbotWeb
}

//go:embed views/*
var Resources embed.FS

func main() {
	gin.SetMode(gin.ReleaseMode)

	wg := sync.WaitGroup{}

	wg.Add(1)
	go func() {
		defer wg.Done()

		router := gin.New()
		router.Use(gin.LoggerWithFormatter(func(param gin.LogFormatterParams) string {
			return fmt.Sprintf("[%s] \"%s %s %s\"\n",
				param.TimeStamp.Format(time.RFC822),
				param.Method,
				param.Path,
				param.ErrorMessage,
			)
		}))
		router.Use(gin.Recovery())

		filebasedDb, err := filebased.InitDB()
		if err != nil {
			panic(err)
		}

		router = RunServer(router, filebasedDb)
		router = RunClient(router, Resources, filebasedDb)

		fmt.Println("Server is running on port 8080")
		err = router.Run(":8080")
		if err != nil {
			panic(err)
		}
	}()

	wg.Wait()
}

func RunServer(router *gin.Engine, filebasedDb *filebased.Data) *gin.Engine {
	userRepo := repo.NewUserRepo(filebasedDb)
	sessionRepo := repo.NewSessionsRepo(filebasedDb)
	categoryRepo := repo.NewCategoryRepo(filebasedDb)
	taskRepo := repo.NewTaskRepo(filebasedDb)

	userService := service.NewUserService(userRepo, sessionRepo)
	categoryService := service.NewCategoryService(categoryRepo)
	taskService := service.NewTaskService(taskRepo)

	userAPIHandler := api.NewUserAPI(userService)
	categoryAPIHandler := api.NewCategoryAPI(categoryService)
	taskAPIHandler := api.NewTaskAPI(taskService)

	apiHandler := APIHandler{
		UserAPIHandler:     userAPIHandler,
		CategoryAPIHandler: categoryAPIHandler,
		TaskAPIHandler:     taskAPIHandler,
	}

	version := router.Group("/api/v1")
	{
		user := version.Group("/user")
		{
			user.POST("/login", apiHandler.UserAPIHandler.Login)
			user.POST("/register", apiHandler.UserAPIHandler.Register)

			user.Use(middleware.Auth())
			user.GET("/tasks", apiHandler.UserAPIHandler.GetUserTaskCategory)
		}

		task := version.Group("/task")
		{
			task.Use(middleware.Auth())
			task.POST("/add", apiHandler.TaskAPIHandler.AddTask)
			task.GET("/get/:id", apiHandler.TaskAPIHandler.GetTaskByID)
			task.PUT("/update/:id", apiHandler.TaskAPIHandler.UpdateTask)
			task.DELETE("/delete/:id", apiHandler.TaskAPIHandler.DeleteTask)
			task.GET("/list", apiHandler.TaskAPIHandler.GetTaskList)
			task.GET("/category/:id", apiHandler.TaskAPIHandler.GetTaskListByCategory)
		}

		category := version.Group("/category")
		{
			category.Use(middleware.Auth())
			category.POST("/add", apiHandler.CategoryAPIHandler.AddCategory)
			category.GET("/get/:id", apiHandler.CategoryAPIHandler.GetCategoryByID)
			category.PUT("/update/:id", apiHandler.CategoryAPIHandler.UpdateCategory)
			category.DELETE("/delete/:id", apiHandler.CategoryAPIHandler.DeleteCategory)
			category.GET("/list", apiHandler.CategoryAPIHandler.GetCategoryList)
		}

		// Add the predict-batik route
		version.POST("/predict-batik", func(c *gin.Context) {
			file, err := c.FormFile("file")
			if err != nil {
				c.JSON(http.StatusBadRequest, gin.H{"error": "Tidak ada file yang diterima"})
				return
			}

			// Save the file locally
			localPath := "./uploads/" + file.Filename
			if err := c.SaveUploadedFile(file, localPath); err != nil {
				c.JSON(http.StatusInternalServerError, gin.H{"error": "Tidak dapat menyimpan file"})
				return
			}

			// Call the prediction API
			prediction, err := predictBatik(localPath)
			if err != nil {
				c.JSON(http.StatusInternalServerError, gin.H{"error": "Tidak dapat mendapatkan prediksi"})
				return
			}

			// Remove the local file
			os.Remove(localPath)

			c.JSON(http.StatusOK, gin.H{"prediction": prediction})
		})
	}

	return router
}

func RunClient(router *gin.Engine, embed embed.FS, filebasedDb *filebased.Data) *gin.Engine {
	sessionRepo := repo.NewSessionsRepo(filebasedDb)
	sessionService := service.NewSessionService(sessionRepo)

	userClient := client.NewUserClient()
	taskClient := client.NewTaskClient()
	categoryClient := client.NewCategoryClient()

	authWeb := web.NewAuthWeb(userClient, sessionService, embed)
	modalWeb := web.NewModalWeb(embed)
	homeWeb := web.NewHomeWeb(embed)
	dashboardWeb := web.NewDashboardWeb(userClient, sessionService, embed)
	taskWeb := web.NewTaskWeb(taskClient, sessionService, embed)
	categoryWeb := web.NewCategoryWeb(categoryClient, sessionService, embed)
	chatbotWeb := web.NewChatbotWeb(embed)

	client := ClientHandler{
		authWeb, homeWeb, dashboardWeb, taskWeb, categoryWeb, modalWeb, chatbotWeb,
	}

	router.StaticFS("/static", http.Dir("frontend/public"))

	router.GET("/", client.HomeWeb.Index)

	user := router.Group("/client")
	{
		user.GET("/login", client.AuthWeb.Login)
		user.POST("/login/process", client.AuthWeb.LoginProcess)
		user.GET("/register", client.AuthWeb.Register)
		user.POST("/register/process", client.AuthWeb.RegisterProcess)

		user.Use(middleware.Auth())
		user.GET("/logout", client.AuthWeb.Logout)
	}

	main := router.Group("/client")
	{
		main.Use(middleware.Auth())
		main.GET("/dashboard", client.DashboardWeb.Dashboard)
		main.GET("/task", client.TaskWeb.TaskPage)
		main.POST("/task/add/process", client.TaskWeb.TaskAddProcess)
		main.GET("/category", client.CategoryWeb.Category)
	}

	modal := router.Group("/client")
	{
		modal.GET("/modal", client.ModalWeb.Modal)
	}

	// Route to handle chatbot interaction
	router.POST("/api/v1/chatbot/interact", client.ChatbotWeb.Interact)

	router.GET("/chatbot", func(c *gin.Context) {
		c.FileFromFS("views/chatbot.html", http.FS(embed))
	})

	router.GET("/Prediksi-batik", func(c *gin.Context) {
		c.FileFromFS("views/Prediksi-batik.html", http.FS(embed))
	})

	return router
}

func predictBatik(filePath string) (string, error) {
	url := "http://127.0.0.1:5000/predict"

	// Open the file
	file, err := os.Open(filePath)
	if err != nil {
		return "", err
	}
	defer file.Close()

	// Create a form that mimics a multipart file upload
	var b bytes.Buffer
	w := multipart.NewWriter(&b)
	fw, err := w.CreateFormFile("image", filepath.Base(file.Name()))
	if err != nil {
		return "", err
	}
	if _, err = io.Copy(fw, file); err != nil {
		return "", err
	}
	w.Close()

	// Create a request
	req, err := http.NewRequest("POST", url, &b)
	if err != nil {
		return "", err
	}
	req.Header.Set("Content-Type", w.FormDataContentType())

	client := &http.Client{}
	res, err := client.Do(req)
	if err != nil {
		return "", err
	}
	defer res.Body.Close()

	if res.StatusCode != http.StatusOK {
		return "", fmt.Errorf("received non-200 response code")
	}

	body, err := ioutil.ReadAll(res.Body)
	if err != nil {
		return "", err
	}

	return string(body), nil
}
