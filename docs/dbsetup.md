# Postgresql Database Setup
Setting up PostgreSQL on Ubuntu involves installing the database server and then performing some basic configuration.
1. Install PostgreSQL:
Update your system's package list.

```bash
sudo apt update
```

Install the PostgreSQL server and its associated tools:

```bash
sudo apt install postgresql postgresql-contrib
```

This command installs the core PostgreSQL server and postgresql-contrib, which provides additional modules and utilities.

2. Verify Installation and Service Status:

Check if the PostgreSQL service is running and enabled to start on boot:

```bash
sudo systemctl status postgresql
```

You should see "active (exited)" or "active (running)" and "enabled".

3. Access the PostgreSQL Shell:

By default, PostgreSQL creates a user named postgres with administrative privileges. Switch to this user to access the psql command-line interface:

```bash
sudo -i -u postgres
```

Enter the PostgreSQL interactive shell.
```bash
psql
```

4. Set a Password for the postgres User (Recommended):

Inside the psql shell, set a password for the postgres user:

```bash
ALTER USER postgres PASSWORD 'your_strong_password';
```

Replace 'your_strong_password' with a secure password. exit the psql shell.
Code

```bash
\q
```

Exit the postgres user session.
```bash
exit
```

5. Create a New Database and User (Optional, but common):
Switch back to the postgres user.
```bash
sudo -i -u postgres
```

Create a new database user (role).
```bash
createuser --interactive
```

Follow the prompts to create a new user, including setting a password.

Create a new database, owned by the new user: 

```bash
createdb your_database_name --owner=your_new_user
```

Exit the postgres user session.
```bash
exit
```