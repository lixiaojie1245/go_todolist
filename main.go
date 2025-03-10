package main

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"time"
	"bufio"

	"github.com/mitchellh/go-homedir"
	"github.com/olekukonko/tablewriter"
	"github.com/urfave/cli/v2"
)

// Task represents a todo task
type Task struct {
	ID          int       `json:"id"`
	Description string    `json:"description"`
	CreatedAt   time.Time `json:"created_at"`
	DueDate     time.Time `json:"due_date"`
	CompletedAt time.Time `json:"completed_at"`
	Completed   bool      `json:"completed"`
	IsDaily     bool      `json:"is_daily"` // New field for daily tasks
}

// TodoList manages a list of tasks
type TodoList struct {
	Tasks []Task `json:"tasks"`
}

// FileStorage handles JSON file operations
type FileStorage struct {
	Filepath string
}

// NewFileStorage creates a new FileStorage instance
func NewFileStorage(filename string) *FileStorage {
	homeDir, _ := homedir.Dir()
	filepath := filepath.Join(homeDir, filename)
	return &FileStorage{Filepath: filepath}
}

// Load loads tasks from JSON file
func (fs *FileStorage) Load() (*TodoList, error) {
	file, err := os.Open(fs.Filepath)
	if err != nil {
		if os.IsNotExist(err) {
			return &TodoList{}, nil
		}
		return nil, err
	}
	defer file.Close()

	var todoList TodoList
	if err := json.NewDecoder(file).Decode(&todoList); err != nil {
		return nil, err
	}
	return &todoList, nil
}

// Save saves tasks to JSON file
func (fs *FileStorage) Save(todoList *TodoList) error {
	file, err := os.Create(fs.Filepath)
	if err != nil {
		return err
	}
	defer file.Close()

	return json.NewEncoder(file).Encode(todoList)
}

// getNextID returns the next available ID
func (tl *TodoList) getNextID() int {
	maxID := 0
	for _, task := range tl.Tasks {
		if task.ID > maxID {
			maxID = task.ID
		}
	}
	return maxID + 1
}

// AddTask adds a new task to the list
func (tl *TodoList) AddTask(description string, dueDate time.Time, isDaily bool) {
	task := Task{
		ID:          tl.getNextID(),
		Description: description,
		CreatedAt:   time.Now(),
		DueDate:     dueDate,
		Completed:   false,
		IsDaily:     isDaily,
	}
	tl.Tasks = append(tl.Tasks, task)
}

// AddDailyTask adds a new daily task
func (tl *TodoList) AddDailyTask(description string) {
	tl.AddTask(description, time.Now().AddDate(0, 0, 1), true)
}

// RemoveDailyTask removes a daily task
func (tl *TodoList) RemoveDailyTask(id int) error {
	for i, task := range tl.Tasks {
		if task.ID == id && task.IsDaily {
			tl.Tasks = append(tl.Tasks[:i], tl.Tasks[i+1:]...)
			return nil
		}
	}
	return fmt.Errorf("daily task not found")
}

// CompleteTask marks a task as completed
func (tl *TodoList) CompleteTask(id int) error {
	for i, task := range tl.Tasks {
		if task.ID == id && !task.Completed {
			tl.Tasks[i].Completed = true
			tl.Tasks[i].CompletedAt = time.Now()
			return nil
		}
	}
	return fmt.Errorf("task not found or already completed")
}

func YesOrNo(remain string,okString string,noOkString string,confirmations []string) bool{

	// 提示用户进行确认操作
	fmt.Print(remain)

	// 获取用户输入
	reader := bufio.NewReader(os.Stdin)
	input, _ := reader.ReadString('\n')

	// 处理换行符和大小写
	input = strings.ToLower(strings.TrimSpace(input))

	// 检查用户输入是否在确认数组中
	isConfirmed := false
	for _, confirmation := range confirmations {
		if input == confirmation {
			isConfirmed = true
			break
		}
	}

	// 根据用户输入执行操作
	if isConfirmed {
		fmt.Println(okString)
		// 在这里执行确认后的操作
		return true
	} else {
		fmt.Println(noOkString)
		// 在这里处理取消或无效输入的情况
		return false
	}
}
// Complete all task
func (tl *TodoList) CompleteAllTask() {
	ok := YesOrNo("This command will completed all task, yes or no? :", 
					"Ok, will completed all",
					"remain all",
					[]string{"yes", "y"},
					)

	if ok {
		for i, task := range tl.Tasks {
			if  !task.Completed {
				tl.Tasks[i].Completed = true
				tl.Tasks[i].CompletedAt = time.Now()
			}
		}
	}
}

// GetIncompleteTasksBeforeDate returns incomplete tasks before a given date
func (tl *TodoList) GetIncompleteTasksBeforeDate(date time.Time) []Task {
	var tasks []Task
	for _, task := range tl.Tasks {
		if !task.Completed && !task.DueDate.After(date) {
			tasks = append(tasks, task)
		}
	}
	return tasks
}

// RefreshDailyTasks refreshes daily tasks for the next day
func (tl *TodoList) RefreshDailyTasks() {
	today := time.Now()
	for i, task := range tl.Tasks {
		if task.IsDaily && !task.Completed && task.DueDate.Before(today) {
			// Create a new instance of the daily task for today
			newTask := Task{
				ID:          tl.getNextID(),
				Description: task.Description,
				CreatedAt:   time.Now(),
				DueDate:     today.AddDate(0, 0, 1),
				Completed:   false,
				IsDaily:     true,
			}
			tl.Tasks = append(tl.Tasks, newTask)
			
			// Mark the old task as completed since it wasn't done yesterday
			tl.Tasks[i].Completed = true
			tl.Tasks[i].CompletedAt = today
		} else if task.IsDaily {
			// Update the due date for daily tasks that are not yet completed
			if task.DueDate.Before(today) {
				tl.Tasks[i].DueDate = today.AddDate(0, 0, 1)
			}
		}
	}
}

// PushIncompleteTasks pushes incomplete tasks to the next day
func (tl *TodoList) PushIncompleteTasks() {
	for i, task := range tl.Tasks {
		if !task.Completed && task.DueDate.Before(time.Now()) && !task.IsDaily {
			tl.Tasks[i].DueDate = tl.Tasks[i].DueDate.AddDate(0, 0, 1)
		}
	}
}
// group 函数将一个切片按照指定的大小进行分组
func group(slice []Task, groupSize int) [][]Task {
    var result [][]Task

    // 如果 groupSize 小于等于0，返回空结果
    if groupSize <= 0 {
        return result
    }

    // 计算需要多少个分组
    numGroups := (len(slice) + groupSize - 1) / groupSize

    // 遍历每个分组
    for i := 0; i < numGroups; i++ {
        // 计算当前分组的起始和结束索引
        start := i * groupSize
        end := start + groupSize

        // 如果结束索引超过了切片长度，调整为切片的长度
        if end > len(slice) {
            end = len(slice)
        }

        // 将当前分组添加到结果中
        result = append(result, slice[start:end])
    }

    return result
}


// 定义一个 map 函数
func mapFunc[T any](slice []T, mapper func(T) T) []T {
    var result []T
    for _, value := range slice {
        result = append(result, mapper(value))
    }
    return result
}

// 定义一个 filter 函数
func filter[T any](slice []T, predicate func(T) bool) []T {
    var result []T
    for _, value := range slice {
        if predicate(value) {
            result = append(result, value)
        }
    }
    return result
}

func MultiHeaderPrintTasks(tasks []Task){
	tasks2D := group(tasks,10)
	for _, tasks1D := range tasks2D {
		PrintTasks(tasks1D)
	}

}
// PrintTasks prints tasks in a table format
func PrintTasks(tasks []Task) {
	if len(tasks) == 0 {
		fmt.Println("No tasks found.")
		return
	}

	table := tablewriter.NewWriter(os.Stdout)
	table.SetHeader([]string{"ID", "Description", "Created At", "Due Date", "Completed At", "Status", "Daily"})

	for _, task := range tasks {
		completedAt := "-"
		if !task.CompletedAt.IsZero() {
			completedAt = task.CompletedAt.Format("2006-01-02 15:04")
		}

		status := "Pending"
		if task.Completed {
			status = "Completed"
		}

		daily := "No"
		if task.IsDaily {
			daily = "Yes"
		}

		table.Append([]string{
			strconv.Itoa(task.ID),
			task.Description,
			task.CreatedAt.Format("2006-01-02 15:04"),
			task.DueDate.Format("2006-01-02"),
			completedAt,
			status,
			daily,
		})
	}

	table.Render()
}

func InitApp() cli.App {
	app := &cli.App{
		Name:    "TodoList",
		Version: "1.0.1",
		Usage:   "A simple TodoList application",
		Commands: []*cli.Command{
			{
				Name:    "add",
				Aliases: []string{"a"},
				Usage:   "Add a new task",
				Flags: []cli.Flag{
					&cli.StringFlag{
						Name:    "description",
						Aliases: []string{"d"},
						Usage:   "Task description",
						Required: true,
					},
					&cli.StringFlag{
						Name:    "due",
						Aliases: []string{"u"},
						Usage:   "Due date (YYYY-MM-DD)",
					},
				},
				Action: func(c *cli.Context) error {
					fs := NewFileStorage(".todolist.json")
					todoList, err := fs.Load()
					if err != nil {
						return err
					}

					description := c.String("description")
					dueDateStr := c.String("due")

					var dueDate time.Time
					if dueDateStr == "" {
						// Default to tomorrow for daily tasks
						dueDate = time.Now().AddDate(0, 0, 1)
					} else {
						dueDate, err = time.Parse("2006-01-02", dueDateStr)
						if err != nil {
							return fmt.Errorf("invalid date format: %v", err)
						}
					}

					todoList.AddTask(description, dueDate, false)
					if err := fs.Save(todoList); err != nil {
						return err
					}

					fmt.Printf("Task added: %s\n", description)
					return nil
				},
			},
			{
				Name:    "add-daily",
				Aliases: []string{"ad"},
				Usage:   "Add a new daily task",
				Flags: []cli.Flag{
					&cli.StringFlag{
						Name:    "description",
						Aliases: []string{"d"},
						Usage:   "Task description",
						Required: true,
					},
				},
				Action: func(c *cli.Context) error {
					fs := NewFileStorage(".todolist.json")
					todoList, err := fs.Load()
					if err != nil {
						return err
					}

					description := c.String("description")
					todoList.AddDailyTask(description)
					if err := fs.Save(todoList); err != nil {
						return err
					}

					fmt.Printf("Daily task added: %s\n", description)
					return nil
				},
			},
			{
				Name:    "remove-daily",
				Aliases: []string{"rd"},
				Usage:   "Remove a daily task",
				Flags: []cli.Flag{
					&cli.IntFlag{
						Name:    "id",
						Aliases: []string{"i"},
						Usage:   "Task ID",
						Required: true,
					},
				},
				Action: func(c *cli.Context) error {
					fs := NewFileStorage(".todolist.json")
					todoList, err := fs.Load()
					if err != nil {
						return err
					}

					id := c.Int("id")
					if err := todoList.RemoveDailyTask(id); err != nil {
						return err
					}

					if err := fs.Save(todoList); err != nil {
						return err
					}

					fmt.Printf("Daily task %d removed\n", id)
					return nil
				},
			},
			{
				Name:    "list-all",
				Aliases: []string{"la"},
				Usage:   "List all tasks",
				Action: func(c *cli.Context) error {
					fs := NewFileStorage(".todolist.json")
					todoList, err := fs.Load()
					if err != nil {
						return err
					}

					// Refresh daily tasks before listing
					todoList.RefreshDailyTasks()
					if err := fs.Save(todoList); err != nil {
						return err
					}

					// PrintTasks(todoList.Tasks)
					MultiHeaderPrintTasks(todoList.Tasks)
					return nil
				},
			},
			{
				Name:    "list",
				Aliases: []string{"l"},
				Usage:   "List tasks when these incompleted",
				Action: func(c *cli.Context) error {
					fs := NewFileStorage(".todolist.json")
					todoList, err := fs.Load()
					if err != nil {
						return err
					}

					// Refresh daily tasks before listing
					todoList.RefreshDailyTasks()
					if err := fs.Save(todoList); err != nil {
						return err
					}

					// PrintTasks(todoList.Tasks)
					incompleted_list:=filter(todoList.Tasks,func(t Task) bool{
						if !t.Completed{
							return true
						}else{
							return false
						}

					})
					// MultiHeaderPrintTasks(todoList.Tasks)
					MultiHeaderPrintTasks(incompleted_list)
					return nil
				},
			},
			{
				Name:    "complete",
				Aliases: []string{"c"},
				Usage:   "Mark a task as completed",
				Flags: []cli.Flag{
					&cli.IntFlag{
						Name:    "id",
						Aliases: []string{"i"},
						Value: -1,
						Usage:   "Task ID",
						Required: false,
					},
				},
				Action: func(c *cli.Context) error {
					fs := NewFileStorage(".todolist.json")
					todoList, err := fs.Load()
					if err != nil {
						return err
					}

					id := c.Int("id")
					if id == -1 {
						todoList.CompleteAllTask()
					}else{
						if err := todoList.CompleteTask(id); err != nil {
							return err
						}
						fmt.Printf("Task %d marked as completed\n", id)
					}

					if err := fs.Save(todoList); err != nil {
						return err
					}

					return nil
					
				},
			},
			{
				Name:    "before",
				Aliases: []string{"b"},
				Usage:   "List tasks due before a specific date",
				Flags: []cli.Flag{
					&cli.StringFlag{
						Name:    "date",
						Aliases: []string{"d"},
						Usage:   "Date (YYYY-MM-DD)",
						Required: true,
					},
				},
				Action: func(c *cli.Context) error {
					fs := NewFileStorage(".todolist.json")
					todoList, err := fs.Load()
					if err != nil {
						return err
					}

					dateStr := c.String("date")
					date, err := time.Parse("2006-01-02", dateStr)
					if err != nil {
						return fmt.Errorf("invalid date format: %v", err)
					}

					tasks := todoList.GetIncompleteTasksBeforeDate(date)
					PrintTasks(tasks)
					return nil
				},
			},
			{
				Name:    "push",
				Aliases: []string{"p"},
				Usage:   "Push incomplete tasks to the next day",
				Action: func(c *cli.Context) error {
					fs := NewFileStorage(".todolist.json")
					todoList, err := fs.Load()
					if err != nil {
						return err
					}

					todoList.PushIncompleteTasks()
					if err := fs.Save(todoList); err != nil {
						return err
					}

					fmt.Println("Incomplete tasks pushed to the next day")
					return nil
				},
			},
		},
	}
	return *app

}
func main() {

	app := InitApp()
	err := app.Run(os.Args)
	if err != nil {
		fmt.Println(err)
		os.Exit(1)
	}
}
