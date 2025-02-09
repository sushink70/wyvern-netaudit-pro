```markdown
# Network Test Dashboard

A simple Django web application that provides a dashboard to perform network tests such as **ping reachability** and **traceroute**. The application features a responsive UI built with Bootstrap and uses AJAX (via Axios) to dynamically display test results without page reloads.

## Features

- **Ping Reachability Test:** Checks if a target device is reachable using the `ping` command.
- **Traceroute Test:** Displays the network path to a target device using the `traceroute` command.
- **Responsive Dashboard:** A clean, user-friendly interface built with Bootstrap.
- **Dynamic Updates:** Results are updated dynamically using AJAX.

## Technologies Used

- **Backend:** Django (Python)
- **Frontend:** Bootstrap 5, HTML, CSS, JavaScript (Axios)
- **System Commands:** `ping` and `traceroute` (executed via Python's subprocess module)

## Prerequisites

- **Python 3.x**
- **pip** (Python package manager)
- **Git** (for version control)
- **System Dependency:** The `traceroute` command must be installed.
  - **Ubuntu/Debian:**
    ```bash
    sudo apt update
    sudo apt install traceroute
    ```
  - **RHEL/CentOS:**
    ```bash
    sudo yum install traceroute
    ```

> **Note:** The provided commands and instructions are tailored for Unix-based systems. Windows users might need to adjust the commands (e.g., using `tracert` instead of `traceroute`).

## Installation

1. **Clone the Repository**
   ```bash
   git clone https://github.com/sushink70/wyvern-netaudit-pro.git
   cd wyvern-netaudit-pro
   ```

2. **Create a Virtual Environment**
   ```bash
   python3 -m venv venv
   ```

3. **Activate the Virtual Environment**
   - On Linux/macOS:
     ```bash
     source venv/bin/activate
     ```
   - On Windows:
     ```bash
     venv\Scripts\activate
     ```

4. **Install Python Dependencies**
   ```bash
   pip install -r requirements.txt
   ```

5. **Set Up System Dependencies**
   Make sure the `traceroute` command is installed on your system (see Prerequisites).

## Usage

1. **Start the Django Development Server**
   ```bash
   python manage.py runserver
   ```

2. **Access the Dashboard**
   Open your browser and navigate to [http://127.0.0.1:8000/](http://127.0.0.1:8000/). You should see the Network Test Dashboard.

3. **Perform Network Tests**
   - **Ping Test:** Enter a target IP or hostname and click the **Ping** button to check its reachability.
   - **Traceroute Test:** Enter a target IP or hostname and click the **Traceroute** button to see the network route.

## Project Structure

```plaintext
my_django_project/
├── venv/                   # Virtual environment (excluded from version control)
├── manage.py               # Django management script
├── netauto/              # Django project configuration
│   ├── __init__.py
│   ├── settings.py
│   ├── urls.py
│   └── wsgi.py
├── netaudit/               # Django app (replace with your app's name)
│   ├── migrations/
│   ├── __init__.py
│   ├── admin.py
│   ├── apps.py
│   ├── models.py
│   ├── tests.py
│   └── views.py
├── templates/              # HTML templates
│   └── dashboard.html      # Dashboard UI template
├── requirements.txt        # Python dependencies list
└── README.md               # This file
```

## Git and Deployment

- **.gitignore:** Ensure that the `venv/` directory, `db.sqlite3`, and other non-essential files are excluded from Git.
- **Pushing Changes to GitHub:**
  After making changes, you can commit and push your updates with:
  ```bash
  git add .
  git commit -m "Describe your changes here"
  git push
  ```

## Contributing

Contributions are welcome! Please fork the repository and create a pull request with your improvements or bug fixes.

## License

This project is licensed under the [MIT License](LICENSE).

## Contact

For any questions or suggestions, please open an issue.
```



