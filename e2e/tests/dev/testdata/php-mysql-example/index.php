<?php

$servername = "mysql";
$database = "mydatabase";
$username = "root";
$password = "mypassword";

// Create connection
$conn = new mysqli($servername, $username, $password);

// Check connection
if ($conn->connect_error) {
    die("Connection failed: " . $conn->connect_error);
} 

// Select devspace database
if (!mysqli_select_db($conn, $database)) {
    die("Uh oh, couldn't select database $database");
}

// Create example table
// sql to create table
$sql = "CREATE TABLE IF NOT EXISTS Users (
    id INT(6) UNSIGNED AUTO_INCREMENT PRIMARY KEY, 
    firstname VARCHAR(255) NOT NULL,
    lastname VARCHAR(255) NOT NULL,
    reg_date TIMESTAMP
)";

if ($conn->query($sql) !== TRUE) {
    die("Error creating table: " . $conn->error);
}

$firstname = null;
$lastname = null;

if(isset($_POST["firstname"]) && isset($_POST["lastname"])) {
    /* create a prepared statement */
    if ($stmt = $conn->prepare("INSERT INTO Users (firstname, lastname, reg_date) VALUES (?, ?, NOW())")) {
        /* bind parameters for markers */
        $stmt->bind_param("ss", $firstname, $lastname);

        $firstname = $_POST["firstname"];
        $lastname = $_POST["lastname"];

        /* execute query */
        $stmt->execute();

        /* close statement */
        $stmt->close();

        header("Location: index.php?user=".$conn->insert_id);
        die();
    } else {
        die("Couldn't create prepared statement: ".$conn->error);
    }
} else if(isset($_GET["user"])) {
    /* create a prepared statement */
    if ($stmt = $conn->prepare("SELECT firstname, lastname FROM Users WHERE id=? LIMIT 1")) {

        /* bind parameters for markers */
        $stmt->bind_param("s", $_GET["user"]);

        /* execute query */
        $stmt->execute();

        /* instead of bind_result: */
        $result = $stmt->get_result();

        /* now you can fetch the results into an array - NICE */
        if ($myrow = $result->fetch_assoc()) {
            $firstname = $myrow['firstname']; 
            $lastname = $myrow['lastname'];
        }

        /* close statement */
        $stmt->close();
    }
} 

?>
<html>
    <head>
        <title><?php $firstname === null ? "Please Register" : "Hello ".$firstname." ".$lastname ?></title>
    </head>
    <body>
        <?php if($firstname == null) : ?>
            <div>
                <div>Please sign in:<div>
                <div>
                    <form action="index.php" method="post">
                        <div>
                            First name: <input type="text" name="firstname"><br>
                        </div>
                        <div>
                            Last name: <input type="text" name="lastname"><br>
                        </div>
                        <div>
                            <input type="submit" value="Submit">
                        </div>
                    </form>
                </div>
            </div>
        <?php else : ?>
            <div>Welcome back <?php echo $firstname." ".$lastname ?></div>
        <?php endif; ?>
    </body>
</html>
