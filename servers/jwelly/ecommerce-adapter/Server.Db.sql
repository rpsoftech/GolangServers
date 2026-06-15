CREATE TABLE
    `ItemGroup` (
        `itemGroupId` INT NOT NULL,
        `itemGroup` VARCHAR(45) NOT NULL,
        `itemPrintName` VARCHAR(45) NULL,
        `itemGroupUnitId` INT NOT NULL,
        PRIMARY KEY (`itemGroupId`)
    );

CREATE TABLE
    `ItemUnit` (
        `itemUnitId` INT NOT NULL,
        `itemUnit` VARCHAR(45) NOT NULL,
        `itemDecimal` INT (1) NULL DEFAULT 0,
        PRIMARY KEY (`itemUnitId`),
        UNIQUE INDEX `itemUnit_UNIQUE` (`itemUnit` ASC) VISIBLE
    );

ALTER TABLE `ItemGroup` ADD INDEX `IgroupUnitToUnitID_idx` (`itemGroupUnitId` ASC) VISIBLE;

ALTER TABLE `ItemGroup` ADD CONSTRAINT `IgroupUnitToUnitID` FOREIGN KEY (`itemGroupUnitId`) REFERENCES `ItemUnit` (`itemUnitId`) ON DELETE RESTRICT ON UPDATE CASCADE;

CREATE TABLE
    `ItemMaster` (
        `itemId` INT NOT NULL,
        `itemName` VARCHAR(45) NULL,
        `iGroupId` INT (3) NULL,
        `itemPrintName` VARCHAR(45) NULL,
        `iUnitId` INT (3) NULL,
        `itemTagPrefix` VARCHAR(5) NULL,
        PRIMARY KEY (`itemId`),
        UNIQUE INDEX `itemName_UNIQUE` (`itemName` ASC) VISIBLE,
        INDEX `ItemMast1_idx` (`iGroupId` ASC) VISIBLE,
        INDEX `ItemMast2_idx` (`iUnitId` ASC) VISIBLE,
        CONSTRAINT `ItemMast1` FOREIGN KEY (`iGroupId`) REFERENCES `ItemGroup` (`itemGroupId`) ON DELETE RESTRICT ON UPDATE CASCADE,
        CONSTRAINT `ItemMast2` FOREIGN KEY (`iUnitId`) REFERENCES `ItemUnit` (`itemUnitId`) ON DELETE RESTRICT ON UPDATE CASCADE
    );

CREATE TABLE
    `Stamp` (
        `stampId` INT NOT NULL,
        `STAMP` VARCHAR(45) NULL,
        `tunch` DECIMAL(8, 2) NULL,
        `stockTunch` DECIMAL(8, 2) NULL,
        `sellTunch` DECIMAL(8, 2) NULL DEFAULT 0,
        PRIMARY KEY (`stampId`)
    );