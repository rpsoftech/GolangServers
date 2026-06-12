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