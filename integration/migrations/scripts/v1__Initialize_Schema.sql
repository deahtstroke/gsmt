CREATE TABLE IF NOT EXISTS department (
    id serial PRIMARY KEY,
    name varchar(255) NOT NULL,
    phoneNumber int NOT NULL
);

CREATE TABLE IF NOT EXISTS employee (
    id serial PRIMARY KEY,
    employeeName text NOT NULL,
    departmentId integer NOT NULL,
    FOREIGN KEY (departmentId) REFERENCES department (id)
);

